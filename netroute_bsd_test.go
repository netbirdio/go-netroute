package netroute

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
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
		gateway:      net.IPv4(127, 0, 0, 1),
		preferredSrc: net.IPv4(127, 0, 0, 1),
		err:          nil,
	},
	// {
	// 	dst:          net.IPv4(127, 0, 0, 0),
	// 	iface:        &net.Interface{Name: "lo0"},
	// 	gateway:      net.IPv4(127, 0, 0, 1),
	// 	preferredSrc: net.IPv4(127, 0, 0, 1),
	// 	err:          nil,
	// },
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
		assert.Equal(t, tt.gateway, gw,  "test %d: expected: %v, got %v", i, tt.gateway, gw)
		assert.Equal(t, tt.preferredSrc, src, "test %d: expected: %v, got %v", i, tt.preferredSrc, src)
	}
}
