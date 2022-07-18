package resolver

import (
	"context"
	"crypto/tls"
	"net"
	"strings"
)

func New(addr string) *net.Resolver {
	switch {
	case strings.HasPrefix(addr, "tls://"):
		return &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
				d := tls.Dialer{}
				address := addr[len("tls://"):]
				return d.DialContext(ctx, "tcp", withDefaultPort(address, "853"))
			},
		}
	case strings.HasPrefix(addr, "https://"):
		return &net.Resolver{
			PreferGo: true,
			Dial: func(_ context.Context, _, _ string) (net.Conn, error) {
				conn := &dohConn{addr: addr}
				return conn, nil
			},
		}
	case addr != "":
		return &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
				d := net.Dialer{}
				address := addr
				network := "udp"

				if strings.HasPrefix(addr, "tcp://") || strings.HasPrefix(addr, "udp://") {
					network = addr[:len("tcp")]
					address = addr[len("tcp://"):]
				}

				return d.DialContext(ctx, network, withDefaultPort(address, "53"))
			},
		}
	default:
		return &net.Resolver{}
	}
}

func withDefaultPort(addr, port string) string {
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr
	}
	return net.JoinHostPort(addr, port)
}
