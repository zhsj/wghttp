package proxy

import (
	"context"
	"fmt"
	"net"
	"testing"
)

func TestDialWithDNS(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	stdDiar := net.Dialer{
		Resolver: &net.Resolver{
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				return nil, fmt.Errorf("dial to %s", address)
			},
		},
	}

	// d := dialWithDNS(stdDiar.DialContext, "https://223.5.5.5")
	d := dialWithDNS(func(ctx context.Context, network, address string) (net.Conn, error) {
		t.Logf("dial to %s:%s", network, address)
		return stdDiar.DialContext(ctx, network, address)
	}, "tls://223.5.5.5")

	for _, addr := range []string{
		"example.com:80",
		"223.6.6.6:80",
		"localhost:22",
		"127.0.0.1:22",
	} {
		t.Run(addr, func(t *testing.T) {
			conn, err := d(context.Background(), "tcp", addr)
			if err != nil {
				t.Error(err)
			} else {
				t.Logf("remote %s", conn.RemoteAddr())
				conn.Close()
			}
		})
	}
}
