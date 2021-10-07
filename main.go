package main

import (
	"encoding/base64"
	"encoding/hex"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/jessevdk/go-flags"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun/netstack"
)

const (
	dns = "1.0.0.1"
	mtu = 1280
)

type options struct {
	PeerEndpoint string   `long:"peer-endpoint" env:"PEER_ENDPOINT" required:"true" description:"WireGuard server address"`
	PeerKey      string   `long:"peer-key" env:"PEER_KEY" required:"true" description:"WireGuard server public key in base64 format"`
	PrivateKey   string   `long:"private-key" env:"PRIVATE_KEY" required:"true" description:"WireGuard client private key in base64 format"`
	ClientIPs    []string `long:"client-ip" env:"CLIENT_IP" env-delim:"," required:"true" description:"WireGuard client address"`
	Listen       string   `long:"listen" env:"LISTEN" required:"true" default:"localhost:8080" description:"http proxy server listen address"`
}

func main() {
	opts := options{}
	parser := flags.NewParser(&opts, flags.Default)
	if _, err := parser.Parse(); err != nil {
		code := 1
		if fe, ok := err.(*flags.Error); ok {
			if fe.Type == flags.ErrHelp {
				code = 0
			}
		}
		os.Exit(code)
	}

	tnet := setupNet(opts)
	ln, err := net.Listen("tcp", opts.Listen)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Listening on %s", ln.Addr())

	s := &http.Server{Handler: httpProxyHandler(tnet.DialContext)}
	if err := s.Serve(ln); err != nil {
		log.Fatal(err)
	}
}

func setupNet(opts options) *netstack.Net {
	privateKey, err := base64.StdEncoding.DecodeString(opts.PrivateKey)
	if err != nil {
		log.Fatalf("Parse private key: %v", err)
	}
	peerKey, err := base64.StdEncoding.DecodeString(opts.PeerKey)
	if err != nil {
		log.Fatalf("Parse peer key: %v", err)
	}
	conf := "private_key=" + hex.EncodeToString(privateKey) + "\n"
	conf += "public_key=" + hex.EncodeToString(peerKey) + "\n"
	conf += "endpoint=" + opts.PeerEndpoint + "\n"
	conf += "allowed_ip=0.0.0.0/0\n"
	conf += "allowed_ip=::/0\n"
	ips := []net.IP{}
	for _, s := range opts.ClientIPs {
		ip := net.ParseIP(s)
		if ip == nil {
			log.Fatalf("invalid local ip: %s", s)
		}
		ips = append(ips, ip)
	}

	tun, tnet, err := netstack.CreateNetTUN(ips, []net.IP{net.ParseIP(dns)}, mtu)
	if err != nil {
		log.Fatal(err)
	}
	dev := device.NewDevice(tun, conn.NewDefaultBind(), device.NewLogger(device.LogLevelError, ""))
	if err := dev.IpcSet(conf); err != nil {
		log.Fatal(err)
	}
	if err := dev.Up(); err != nil {
		log.Fatal(err)
	}
	return tnet
}
