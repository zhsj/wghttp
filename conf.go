package main

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"time"

	"github.com/zhsj/wghttp/internal/resolver"
	"golang.zx2c4.com/wireguard/device"
)

type peer struct {
	dialer *net.Dialer

	pubKey string
	psk    string

	addr   string
	ipPort string
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
		dialer: &net.Dialer{
			Resolver: resolver.New(opts.ResolveDNS),
		},
		pubKey: hex.EncodeToString(pubKey),
		psk:    hex.EncodeToString(psk),
		addr:   opts.PeerEndpoint,
	}
	p.ipPort, err = p.resolveAddr()
	if err != nil {
		return nil, fmt.Errorf("resolve peer endpoint: %w", err)
	}
	return p, err
}

func (p *peer) initConf() string {
	conf := "public_key=" + p.pubKey + "\n"
	conf += "endpoint=" + p.ipPort + "\n"
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
	newIPPort, err := p.resolveAddr()
	if err != nil {
		logger.Verbosef("Resolve peer endpoint: %v", err)
		return "", false
	}
	if p.ipPort == newIPPort {
		return "", false
	}
	p.ipPort = newIPPort
	logger.Verbosef("PeerEndpoint is changed to: %s", p.ipPort)

	conf := "public_key=" + p.pubKey + "\n"
	conf += "update_only=true\n"
	conf += "endpoint=" + p.ipPort + "\n"
	return conf, true
}

func (p *peer) resolveAddr() (string, error) {
	c, err := p.dialer.Dial("udp", p.addr)
	if err != nil {
		return "", err
	}
	defer c.Close()
	return c.RemoteAddr().String(), nil
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

	if peer.addr != peer.ipPort {
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
