package proxy

import (
	"context"
	"encoding/json"
	"net"
	"net/http"

	"github.com/zhsj/wghttp/internal/resolver"
	"github.com/zhsj/wghttp/internal/third_party/tailscale/httpproxy"
	"github.com/zhsj/wghttp/internal/third_party/tailscale/proxymux"
	"github.com/zhsj/wghttp/internal/third_party/tailscale/socks5"
)

type dialer func(ctx context.Context, network, address string) (net.Conn, error)

type Proxy struct {
	Dial  dialer
	DNS   string
	Stats func() (any, error)
}

func statsHandler(next http.Handler, stats func() (any, error)) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.URL.Host != "" || r.URL.Path != "/stats" {
			next.ServeHTTP(rw, r)
			return
		}
		s, err := stats()
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
		} else {
			resp, _ := json.MarshalIndent(s, "", "  ")
			rw.Header().Set("Content-Type", "application/json")
			_, _ = rw.Write(append(resp, '\n'))
		}
	})
}

func dialWithDNS(dial dialer, dns string) dialer {
	resolv := resolver.New(dns, dial)

	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		if err == nil {
			if ip := net.ParseIP(host); ip != nil {
				return dial(ctx, network, address)
			}
		}

		ips, err := resolv.LookupHost(ctx, host)
		if err != nil {
			return nil, err
		}

		var (
			lastErr error
			conn    net.Conn
		)
		for _, ip := range ips {
			addr := net.JoinHostPort(ip, port)
			conn, lastErr = dial(ctx, network, addr)
			if lastErr == nil {
				return conn, nil
			}
		}
		return nil, lastErr
	}
}

func (p Proxy) Serve(ln net.Listener) {
	d := dialWithDNS(p.Dial, p.DNS)

	socksListener, httpListener := proxymux.SplitSOCKSAndHTTP(ln)

	httpProxy := &http.Server{Handler: statsHandler(httpproxy.Handler(d), p.Stats)}
	socksProxy := &socks5.Server{Dialer: d}

	errc := make(chan error, 2)
	go func() {
		if err := httpProxy.Serve(httpListener); err != nil {
			errc <- err
		}
	}()
	go func() {
		if err := socksProxy.Serve(socksListener); err != nil {
			errc <- err
		}
	}()
	<-errc
}
