package main

import (
	"bufio"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strings"

	"github.com/jessevdk/go-flags"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun/netstack"

	"github.com/zhsj/wghttp/internal/proxy"
)

//go:embed README.md
var readme string

var (
	logger *device.Logger
	opts   options
)

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
		fe := &flags.Error{}
		if errors.As(err, &fe) && fe.Type == flags.ErrHelp {
			code = 0
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

	proxier := proxy.Proxy{
		Dial: proxyDialer(tnet), DNS: opts.DNS, Stats: stats(dev),
	}
	proxier.Serve(listener)

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
	clientIPs := []netip.Addr{}
	for _, ip := range opts.ClientIPs {
		clientIPs = append(clientIPs, netip.Addr(ip))
	}
	tun, tnet, err := netstack.CreateNetTUN(clientIPs, nil, opts.MTU)
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
