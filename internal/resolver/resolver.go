package resolver

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

var errNotRetry = errors.New("not retry")

type Resolver struct {
	sysAddr, addr string
	network       string
	tlsConfig     *tls.Config
	httpClient    *http.Client

	r *net.Resolver
}

func (r *Resolver) LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) {
	ipNetwork := network
	switch network {
	case "tcp", "udp":
		ipNetwork = "ip"
	case "tcp4", "udp4":
		ipNetwork = "ip4"
	case "tcp6", "udp6":
		ipNetwork = "ip6"
	}

	return r.r.LookupNetIP(ctx, ipNetwork, host)
}

func New(dns string, dial func(ctx context.Context, network, address string) (net.Conn, error)) *Resolver {
	r := &Resolver{}
	switch {
	case strings.HasPrefix(dns, "tls://"):
		r.addr = withDefaultPort(dns[len("tls://"):], "853")
		host, _, _ := net.SplitHostPort(r.addr)
		r.tlsConfig = &tls.Config{
			ServerName: host,
		}
		r.r = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, _, address string) (net.Conn, error) {
				if r.sysAddr == "" {
					r.sysAddr = address
				}
				if r.sysAddr != address {
					return nil, errNotRetry
				}
				conn, err := dial(ctx, "tcp", r.addr)
				if err != nil {
					return nil, err
				}
				return tls.Client(conn, r.tlsConfig), nil
			},
		}
	case strings.HasPrefix(dns, "https://"):
		r.httpClient = &http.Client{
			Transport: &http.Transport{
				DialContext: dial,
			},
		}
		r.r = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, _, address string) (net.Conn, error) {
				if r.sysAddr == "" {
					r.sysAddr = address
				}
				if r.sysAddr != address {
					return nil, errNotRetry
				}

				return newDoHConn(ctx, r.httpClient, dns)
			},
		}
	case dns != "":
		r.addr = dns
		r.network = "udp"

		if strings.HasPrefix(dns, "tcp://") || strings.HasPrefix(dns, "udp://") {
			r.addr = dns[len("tcp://"):]
			r.network = dns[:len("tcp")]
		}
		r.addr = withDefaultPort(r.addr, "53")

		r.r = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, _, address string) (net.Conn, error) {
				if r.sysAddr == "" {
					r.sysAddr = address
				}
				if r.sysAddr != address {
					return nil, errNotRetry
				}

				return dial(ctx, r.network, r.addr)
			},
		}
	default:
		r.r = &net.Resolver{}
	}
	return r
}

func withDefaultPort(addr, port string) string {
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr
	}
	return net.JoinHostPort(addr, port)
}
