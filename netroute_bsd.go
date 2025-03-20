// Copyright 2012 Google, Inc. All rights reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

//go:build darwin || dragonfly || freebsd || netbsd || openbsd

// This is a BSD import for the routing structure initially found in
// https://github.com/google/gopacket/blob/master/routing/routing.go
// RIB parsing follows the BSD route format described in
// https://github.com/freebsd/freebsd/blob/master/sys/net/route.h
package netroute

import (
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"

	"github.com/google/gopacket/routing"
	"golang.org/x/net/route"
	"golang.org/x/sys/unix"
)

const (
	RTF_IFSCOPE = 0x1000000
)

type bsdRouter struct{}

func toIPAddr(a route.Addr) net.IP {
	switch t := a.(type) {
	case *route.Inet4Addr:
		return t.IP[:]
	case *route.Inet6Addr:
		return t.IP[:]
	default:
		return nil
	}
}

// toRouteAddr converts the net.IP to route.Addr .
// If ip is not an IPv4 address, toRouteAddr returns nil.
func toRouteAddr(ip net.IP) route.Addr {
	if len(ip) == 0 {
		return nil
	}

	if len(ip) != net.IPv4len && len(ip) != net.IPv6len {
		return nil
	}
	if p4 := ip.To4(); len(p4) == net.IPv4len {
		return &route.Inet4Addr{IP: [4]byte(p4)}
	}
	return &route.Inet6Addr{IP: [16]byte(ip)}
}

// IPToIfIndex takes an IP and returns index of the interface with the given IP assigned if any
func IPToIfIndex(ip net.IP) (int, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return -1, fmt.Errorf("failed to get interfaces: %s", err)
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			inet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			if inet.IP.String() == ip.String() {
				return iface.Index, nil
			}
		}
	}
	return -1, fmt.Errorf("no interface found for IP: %s", ip)
}

func (r *bsdRouter) Route(dst net.IP) (iface *net.Interface, gateway, preferredSrc net.IP, err error) {
	return r.RouteWithSrc(nil, nil, dst)
}

func (r *bsdRouter) RouteWithSrc(input net.HardwareAddr, src, dst net.IP) (iface *net.Interface, gateway, preferredSrc net.IP, err error) {
	var mFlags int
	var mIndex int
	dstAddr := toRouteAddr(dst)
	if dstAddr == nil {
		return nil, nil, nil, fmt.Errorf("failed to parse dst: %#v", dst)
	}
	if src != nil {
		i, err := IPToIfIndex(src)
		if err != nil {
			return nil, nil, nil, err
		}
		mIndex = i
		mFlags = RTF_IFSCOPE
	}

	pid := os.Getpid()
	m := &route.RouteMessage{
		Version: syscall.RTM_VERSION,
		Type:    unix.RTM_GET,
		Index:   mIndex,
		Flags:   mFlags,
		ID:      uintptr(pid),
		Seq:     0,
		Addrs: []route.Addr{
			dstAddr,
			nil,
			nil,
			nil,
			&route.LinkAddr{},
		},
	}
	var rm *route.RouteMessage
	if rm, err = GetRouteMsgAnswer(m); err != nil {
		return
	}
	if iface, err = net.InterfaceByIndex(rm.Index); err != nil {
		return
	}
	preferredSrc = toIPAddr(rm.Addrs[5])
	if dst.String() == preferredSrc.String() {
		return
	}
	gateway = toIPAddr(rm.Addrs[1])
	return
}

// GetRouteMsgAnswer takes an RTM_GET RouteMessage and returns a answer (RouteMessage)
func GetRouteMsgAnswer(m *route.RouteMessage) (*route.RouteMessage, error) {
	if m.Type != syscall.RTM_GET {
		return nil, errors.New("message type is not RTM_GET")
	}
	so, err := unix.Socket(unix.AF_ROUTE, unix.SOCK_RAW, unix.AF_UNSPEC)
	if err != nil {
		return nil, err
	}
	defer unix.Close(so)

	wb, err := m.Marshal()
	if err != nil {
		return nil, err
	}
	if _, err := unix.Write(so, wb); err != nil {
		return nil, err
	}

	var rb [2 << 10]byte
	for {
		n, err := unix.Read(so, rb[:])
		if err != nil {
			return nil, fmt.Errorf("failed to read from routing socket: %s", err)
		}
		// Parse the response messages
		rms, err := route.ParseRIB(route.RIBTypeRoute, rb[:n])
		if err != nil {
			return nil, fmt.Errorf("failed to parsed messages: %s", err)
		}
		if len(rms) == 0 {
			break
		}
		rm, ok := rms[0].(*route.RouteMessage)
		// confirm it is a reply to our query
		if !ok || (m.ID != rm.ID && m.Seq != rm.Seq && rm.Type != unix.RTM_GET) {
			continue
		}
		return rm, nil
	}
	return nil, errors.New("failed to read from routing socket")
}

func New() (routing.Router, error) {
	rtr := &bsdRouter{}
	return rtr, nil
}
