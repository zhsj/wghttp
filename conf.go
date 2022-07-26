package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"time"

	"github.com/zhsj/wghttp/internal/resolver"
	"golang.zx2c4.com/wireguard/device"
)

type peer struct {
	resolver *resolver.Resolver

	pubKey string
	psk    string

	host string
	ip   netip.Addr
	port uint16
}

func newPeerEndpoint() (*peer, error) {
	pubKey, err := base64.StdEncoding.DecodeString(opts.PeerKey)
	if err != nil {
		return nil, fmt.Errorf("parse peer public key: %w", err)
	}
	psk, err := base64.StdEncoding.DecodeString(opts.PresharedKey)
	if err != nil {
		return nil, fmt.Errorf("parse preshared key: %w", err)
	}

	p := &peer{
		pubKey: hex.EncodeToString(pubKey),
		psk:    hex.EncodeToString(psk),
	}
	host, port, err := net.SplitHostPort(opts.PeerEndpoint)
	if err != nil {
		return nil, fmt.Errorf("parse peer endpoint: %w", err)
	}
	port16, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("parse peer endpoint port: %w", err)
	}
	p.host = host
	p.port = uint16(port16)

	p.ip, err = netip.ParseAddr(p.host)
	if err == nil {
		return p, nil
	}

	p.resolver = resolver.New(
		opts.ResolveDNS,
		func(ctx context.Context, network, address string) (net.Conn, error) {
			netConn, err := (&net.Dialer{}).DialContext(ctx, network, address)
			logger.Verbosef("Using %s to resolve peer endpoint: %v", opts.ResolveDNS, err)
			return netConn, err
		},
	)

	p.ip, err = p.resolveHost()
	if err != nil {
		return nil, fmt.Errorf("resolve peer endpoint ip: %w", err)
	}

	return p, err
}

func (p *peer) initConf() string {
	conf := "public_key=" + p.pubKey + "\n"
	conf += "endpoint=" + netip.AddrPortFrom(p.ip, p.port).String() + "\n"
	conf += "allowed_ip=0.0.0.0/0\n"
	conf += "allowed_ip=::/0\n"

	if opts.KeepaliveInterval > 0 {
		conf += fmt.Sprintf("persistent_keepalive_interval=%.f\n", opts.KeepaliveInterval.Seconds())
	}
	if p.psk != "" {
		conf += "preshared_key=" + p.psk + "\n"
	}

	return conf
}

func (p *peer) updateConf() (string, bool) {
	newIP, err := p.resolveHost()
	if err != nil {
		logger.Verbosef("Resolve peer endpoint: %v", err)
		return "", false
	}
	if p.ip == newIP {
		return "", false
	}
	p.ip = newIP
	logger.Verbosef("PeerEndpoint is changed to: %s", p.ip)

	conf := "public_key=" + p.pubKey + "\n"
	conf += "update_only=true\n"
	conf += "endpoint=" + netip.AddrPortFrom(p.ip, p.port).String() + "\n"
	return conf, true
}

func (p *peer) resolveHost() (netip.Addr, error) {
	ips, err := p.resolver.LookupNetIP(context.Background(), "ip", p.host)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("resolve ip for %s: %w", p.host, err)
	}
	for _, ip := range ips {
		// netstack doesn't seem to understand IPv4-mapped IPv6 addresses.
		ip = ip.Unmap()
		conn, err := net.DialUDP("udp", nil, net.UDPAddrFromAddrPort(netip.AddrPortFrom(ip, p.port)))
		if err == nil {
			conn.Close()
			return ip, nil
		} else {
			logger.Verbosef("Dial %s: %s", ip, err)
		}
	}
	return netip.Addr{}, fmt.Errorf("no available ip for %s", p.host)
}

func ipcSet(dev *device.Device) error {
	privateKey, err := base64.StdEncoding.DecodeString(opts.PrivateKey)
	if err != nil {
		return fmt.Errorf("parse client private key: %w", err)
	}
	conf := "private_key=" + hex.EncodeToString(privateKey) + "\n"
	if opts.ClientPort != 0 {
		conf += fmt.Sprintf("listen_port=%d\n", opts.ClientPort)
	}

	peer, err := newPeerEndpoint()
	if err != nil {
		return err
	}
	conf += peer.initConf()
	logger.Verbosef("Device config:\n%s", conf)

	if err := dev.IpcSet(conf); err != nil {
		return err
	}

	if peer.resolver != nil {
		go func() {
			c := time.Tick(opts.ResolveInterval)

			for range c {
				conf, needUpdate := peer.updateConf()
				if !needUpdate {
					continue
				}

				if err := dev.IpcSet(conf); err != nil {
					logger.Errorf("Config device: %v", err)
				}
			}
		}()
	}
	return nil
}
