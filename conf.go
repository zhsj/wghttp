package main

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"time"

	"golang.zx2c4.com/wireguard/device"
)

func ipcSet(dev *device.Device, opts options) error {
	privateKey, err := base64.StdEncoding.DecodeString(opts.PrivateKey)
	if err != nil {
		return fmt.Errorf("parse private key: %w", err)
	}
	peerKey, err := base64.StdEncoding.DecodeString(opts.PeerKey)
	if err != nil {
		return fmt.Errorf("parse peer key: %w", err)
	}
	conf := "private_key=" + hex.EncodeToString(privateKey) + "\n"
	conf += "public_key=" + hex.EncodeToString(peerKey) + "\n"

	peerAddr, err := net.ResolveUDPAddr("udp", opts.PeerEndpoint)
	if err != nil {
		return fmt.Errorf("resolve peer endpoint: %w", err)
	}

	conf += "endpoint=" + peerAddr.String() + "\n"
	conf += "allowed_ip=0.0.0.0/0\n"
	conf += "allowed_ip=::/0\n"

	if opts.ExitMode == "local" {
		conf += "persistent_keepalive_interval=10\n"
	}

	if err := dev.IpcSet(conf); err != nil {
		return fmt.Errorf("set device config: %w", err)
	}

	if peerAddr.String() != opts.PeerEndpoint {
		go refreshEndpoint(dev, peerKey, peerAddr.String(), opts.PeerEndpoint)
	}
	return nil
}

func refreshEndpoint(dev *device.Device, peerKey []byte, currentPeerAddr, peerEndpoint string) {
	c := time.Tick(10 * time.Second)

	for range c {
		addr, err := net.ResolveUDPAddr("udp", peerEndpoint)
		if err != nil {
			logger.Errorf("Resolve peer endpoint: %v", err)
			continue
		}
		if currentPeerAddr == addr.String() {
			continue
		}
		currentPeerAddr = addr.String()
		logger.Verbosef("Endpoint is changed to: %s", addr)
		conf := "public_key=" + hex.EncodeToString(peerKey) + "\n"
		conf += "update_only=true\n"
		conf += "endpoint=" + addr.String() + "\n"
		if err := dev.IpcSet(conf); err != nil {
			logger.Errorf("Set device config: %v", err)
		}
	}
}
