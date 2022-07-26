package main

import (
	"encoding/base64"
	"encoding/hex"
	"net"
	"net/netip"
	"strconv"
	"time"
)

type ipT netip.Addr

func (o *ipT) UnmarshalFlag(value string) error {
	ip, err := netip.ParseAddr(value)
	*o = ipT(ip)
	return err
}

func (o ipT) String() string {
	return netip.Addr(o).String()
}

type hostPortT struct {
	host string
	port uint16
}

func (o *hostPortT) UnmarshalFlag(value string) error {
	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return err
	}
	port16, err := strconv.ParseUint(port, 10, 16)
	*o = hostPortT{host, uint16(port16)}
	return err
}

type keyT string

func (o *keyT) UnmarshalFlag(value string) error {
	key, err := base64.StdEncoding.DecodeString(value)
	*o = keyT(hex.EncodeToString(key))
	return err
}

type timeT int64

func (o *timeT) UnmarshalFlag(value string) error {
	i, err := strconv.ParseInt(value, 10, 32)
	if err == nil {
		*o = timeT(i)
		return nil
	}
	d, err := time.ParseDuration(value)
	*o = timeT(d.Seconds())
	return err
}

type options struct {
	ClientIPs  []ipT  `long:"client-ip" env:"CLIENT_IP" env-delim:"," required:"true" description:"[Interface].Address\tfor WireGuard client (can be set multiple times)"`
	ClientPort int    `long:"client-port" env:"CLIENT_PORT" description:"[Interface].ListenPort\tfor WireGuard client (optional)"`
	PrivateKey keyT   `long:"private-key" env:"PRIVATE_KEY" required:"true" description:"[Interface].PrivateKey\tfor WireGuard client (format: base64)"`
	DNS        string `long:"dns" env:"DNS" description:"[Interface].DNS\tfor WireGuard network (format: protocol://ip:port)\nProtocol includes udp(default), tcp, tls(DNS over TLS) and https(DNS over HTTPS)"`
	MTU        int    `long:"mtu" env:"MTU" default:"1280" description:"[Interface].MTU\tfor WireGuard network"`

	PeerEndpoint      hostPortT `long:"peer-endpoint" env:"PEER_ENDPOINT" required:"true" description:"[Peer].Endpoint\tfor WireGuard server (format: host:port)"`
	PeerKey           keyT      `long:"peer-key" env:"PEER_KEY" required:"true" description:"[Peer].PublicKey\tfor WireGuard server (format: base64)"`
	PresharedKey      keyT      `long:"preshared-key" env:"PRESHARED_KEY" description:"[Peer].PresharedKey\tfor WireGuard network (optional, format: base64)"`
	KeepaliveInterval timeT     `long:"keepalive-interval" env:"KEEPALIVE_INTERVAL" description:"[Peer].PersistentKeepalive\tfor WireGuard network (optional)"`

	ResolveDNS      string `long:"resolve-dns" env:"RESOLVE_DNS" description:"DNS for resolving WireGuard server address (optional, format: protocol://ip:port)\nProtocol includes udp(default), tcp, tls(DNS over TLS) and https(DNS over HTTPS)"`
	ResolveInterval timeT  `long:"resolve-interval" env:"RESOLVE_INTERVAL" default:"1m" description:"Interval for resolving WireGuard server address (set 0 to disable)"`

	Listen   string `long:"listen" env:"LISTEN" default:"localhost:8080" description:"HTTP & SOCKS5 server address"`
	ExitMode string `long:"exit-mode" env:"EXIT_MODE" choice:"remote" choice:"local" default:"remote" description:"Exit mode"`
	Verbose  bool   `short:"v" long:"verbose" description:"Show verbose debug information"`

	ClientID string `long:"client-id" env:"CLIENT_ID" hidden:"true"`
}
