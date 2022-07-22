package resolver

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strings"
)

// PreferGo works on Windows since go1.19, https://github.com/golang/go/issues/33097

func New(addr string, dialContext func(context.Context, string, string) (net.Conn, error)) *net.Resolver {
	switch {
	case strings.HasPrefix(addr, "tls://"):
		return &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
				address := withDefaultPort(addr[len("tls://"):], "853")
				conn, err := dialContext(ctx, "tcp", address)
				if err != nil {
					return nil, err
				}
				host, _, _ := net.SplitHostPort(address)
				c := tls.Client(conn, &tls.Config{
					ServerName: host,
				})
				return c, nil
			},
		}
	case strings.HasPrefix(addr, "https://"):
		c := &http.Client{
			Transport: &http.Transport{
				DialContext: dialContext,
			},
		}
		return &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return newDoHConn(ctx, c, addr)
			},
		}
	case addr != "":
		return &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
				address := addr
				network := "udp"

				if strings.HasPrefix(addr, "tcp://") || strings.HasPrefix(addr, "udp://") {
					network = addr[:len("tcp")]
					address = addr[len("tcp://"):]
				}

				return dialContext(ctx, network, withDefaultPort(address, "53"))
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
