//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package netroute

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/route"
)

type routeOnBSDTest struct {
	dst                   net.IP
	iface                 *net.Interface
	gateway, preferredSrc net.IP
	err                   error
}

var routeOnBSDTests = []routeOnBSDTest{
	{
		dst:          net.IPv4(127, 0, 0, 1),
		iface:        &net.Interface{Name: "lo0"},
		gateway:      nil,
		preferredSrc: net.IPv4(127, 0, 0, 1),
		err:          nil,
	},
}

func TestRouteOnBSD(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatal(err)
	}
	for i, tt := range routeOnBSDTests {
		iface, gw, src, err := r.Route(tt.dst)
		if tt.err != nil {
			assert.NoError(t, err, "test %d: got error, none expected", i)
		}
		assert.Equal(t, tt.iface.Name, iface.Name, "test %d: expected: %v, got %v", i, tt.iface.Name, iface.Name)
		assert.Equal(t, tt.preferredSrc, src, "test %d: expected: %v, got %v", i, tt.preferredSrc, src)
		assert.Equal(t, tt.gateway, gw, "test %d: expected: %v, got %v", i, tt.gateway, gw)
	}
}

type ipToIfIndexTest struct {
	ip    net.IP
	index int
}

var ipToIfIndexTests = []ipToIfIndexTest{
	{
		net.IPv4(127, 0, 0, 1),
		1,
	},
	{
		net.IPv4(127, 0, 0, 100),
		-1,
	},
}

func TestIPToIfIndex(t *testing.T) {
	for i, tt := range ipToIfIndexTests {
		ifIndex, err := IPToIfIndex(tt.ip)
		if tt.index >= 1 {
			assert.NoError(t, err, "test %d: got error, none expected", i)
			assert.GreaterOrEqual(t, ifIndex, 1)
			continue
		}
		assert.LessOrEqual(t, ifIndex, 0)
	}
}

type toRouteAddrTest struct {
	ip      net.IP
	addr    route.Addr
	passing bool
}

var toRouteAddrTests = []toRouteAddrTest{
	{
		net.IPv4(127, 0, 0, 1),
		&route.Inet4Addr{IP: [4]byte{127, 0, 0, 1}},
		true,
	},
	{
		net.IPv4(127, 0, 0, 1),
		&route.Inet4Addr{IP: [4]byte{127, 0, 0, 0}},
		false,
	},
}

func TestToRoueAddr(t *testing.T) {
	for i, tt := range toRouteAddrTests {
		a := toRouteAddr(tt.ip)
		if tt.passing {
			assert.Equal(t, tt.addr, a, "test %d: expected: %v, got %v", i, tt.addr, a)
			continue
		}
		assert.NotEqual(t, tt.addr, a, "test %d: did not expect: %v, but got %v", i, tt.addr, a)
	}
}
