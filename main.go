package main

import (
	"bufio"
	"context"
	_ "embed"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jessevdk/go-flags"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun/netstack"

	"github.com/zhsj/wghttp/internal/third_party/tailscale/httpproxy"
	"github.com/zhsj/wghttp/internal/third_party/tailscale/proxymux"
	"github.com/zhsj/wghttp/internal/third_party/tailscale/socks5"
)

//go:embed README.md
var readme string

var (
	logger *device.Logger
	opts   options
)

type options struct {
	PeerEndpoint string   `long:"peer-endpoint" env:"PEER_ENDPOINT" description:"WireGuard server address"`
	PeerKey      string   `long:"peer-key" env:"PEER_KEY" description:"WireGuard server public key in base64 format"`
	PrivateKey   string   `long:"private-key" env:"PRIVATE_KEY" description:"WireGuard client private key in base64 format"`
	ClientIPs    []string `long:"client-ip" env:"CLIENT_IP" env-delim:"," description:"WireGuard client IP address"`

	DNS string `long:"dns" env:"DNS" description:"DNS server for WireGuard network and resolving server address"`
	DoT string `long:"dot" env:"DOT" description:"Port for DNS over TLS, used to resolve WireGuard server address if available"`
	MTU int    `long:"mtu" env:"MTU" default:"1280" description:"MTU for WireGuard network"`

	KeepaliveInterval time.Duration `long:"keepalive-interval" env:"KEEPALIVE_INTERVAL" description:"Interval for sending keepalive packet"`
	ResolveInterval   time.Duration `long:"resolve-interval" env:"RESOLVE_INTERVAL" default:"1m" description:"Interval for resolving WireGuard server address"`

	Listen   string `long:"listen" env:"LISTEN" default:"localhost:8080" description:"HTTP & SOCKS5 server address"`
	ExitMode string `long:"exit-mode" env:"EXIT_MODE" choice:"remote" choice:"local" default:"remote" description:"Exit mode"`
	Verbose  bool   `short:"v" long:"verbose" description:"Show verbose debug information"`

	ClientID string `long:"client-id" env:"CLIENT_ID" hidden:"true"`
}

func main() {
	parser := flags.NewParser(&opts, flags.Default)
	parser.Usage = `[OPTIONS]

Description:`
	scanner := bufio.NewScanner(strings.NewReader(strings.TrimPrefix(readme, "# wghttp\n")))
	for scanner.Scan() {
		parser.Usage += "  " + scanner.Text() + "\n"
	}
	parser.Usage = strings.TrimSuffix(parser.Usage, "\n")
	if _, err := parser.Parse(); err != nil {
		code := 1
		if fe, ok := err.(*flags.Error); ok {
			if fe.Type == flags.ErrHelp {
				code = 0
			}
		}
		os.Exit(code)
	}
	if opts.Verbose {
		logger = device.NewLogger(device.LogLevelVerbose, "")
	} else {
		logger = device.NewLogger(device.LogLevelError, "")
	}
	logger.Verbosef("Options: %+v", opts)

	dev, tnet, err := setupNet()
	if err != nil {
		logger.Errorf("Setup netstack: %v", err)
		os.Exit(1)
	}

	listener, err := proxyListener(tnet)
	if err != nil {
		logger.Errorf("Create net listener: %v", err)
		os.Exit(1)
	}

	socksListener, httpListener := proxymux.SplitSOCKSAndHTTP(listener)
	dialer := proxyDialer(tnet)

	httpProxy := &http.Server{Handler: statsHandler(httpproxy.Handler(dialer), dev)}
	socksProxy := &socks5.Server{Dialer: dialer}

	errc := make(chan error, 2)
	go func() {
		if err := httpProxy.Serve(httpListener); err != nil {
			logger.Errorf("Serving http proxy: %v", err)
			errc <- err
		}
	}()
	go func() {
		if err := socksProxy.Serve(socksListener); err != nil {
			logger.Errorf("Serving socks5 proxy: %v", err)
			errc <- err
		}
	}()
	<-errc
	os.Exit(1)
}

func proxyDialer(tnet *netstack.Net) (dialer func(ctx context.Context, network, address string) (net.Conn, error)) {
	switch opts.ExitMode {
	case "local":
		d := net.Dialer{}
		dialer = d.DialContext
	case "remote":
		dialer = tnet.DialContext
	}
	return
}

func proxyListener(tnet *netstack.Net) (net.Listener, error) {
	var tcpListener net.Listener

	tcpAddr, err := net.ResolveTCPAddr("tcp", opts.Listen)
	if err != nil {
		return nil, fmt.Errorf("resolve listen addr: %w", err)
	}

	switch opts.ExitMode {
	case "local":
		tcpListener, err = tnet.ListenTCP(tcpAddr)
		if err != nil {
			return nil, fmt.Errorf("create listener on netstack: %w", err)
		}
	case "remote":
		tcpListener, err = net.ListenTCP("tcp", tcpAddr)
		if err != nil {
			return nil, fmt.Errorf("create listener on local net: %w", err)
		}
	}
	logger.Verbosef("Listening on %s", tcpListener.Addr())
	return tcpListener, nil
}

func setupNet() (*device.Device, *netstack.Net, error) {
	ips := []net.IP{}
	for _, s := range opts.ClientIPs {
		if ip := net.ParseIP(s); ip != nil {
			ips = append(ips, ip)
		}
	}
	if len(ips) == 0 {
		return nil, nil, fmt.Errorf("not have a valid client ip: %s", opts.ClientIPs)
	}
	dnsServers := []net.IP{}
	if ip := net.ParseIP(opts.DNS); ip != nil {
		dnsServers = append(dnsServers, ip)
	}
	tun, tnet, err := netstack.CreateNetTUN(ips, dnsServers, opts.MTU)
	if err != nil {
		return nil, nil, fmt.Errorf("create netstack tun: %w", err)
	}
	dev := device.NewDevice(tun, newConnBind(opts.ClientID), logger)

	if err := ipcSet(dev); err != nil {
		return nil, nil, fmt.Errorf("config device: %w", err)
	}

	if err := dev.Up(); err != nil {
		return nil, nil, fmt.Errorf("bring up device: %w", err)
	}

	return dev, tnet, nil
}
