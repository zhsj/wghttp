package resolver

import (
	"context"
	"net"
	"testing"
)

func TestResolve(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	for _, server := range []string{
		"",
		"223.5.5.5",
		"223.5.5.5:53",
		"tcp://223.5.5.5",
		"tcp://223.5.5.5:53",
		"udp://223.5.5.5",
		"udp://223.5.5.5:53",
		"tls://223.5.5.5",
		"tls://223.5.5.5:853",
		"https://223.5.5.5",
		"https://223.5.5.5:443",
		"https://223.5.5.5:443/dns-query",
	} {
		t.Run(server, func(t *testing.T) {
			r := New(server, (&net.Dialer{}).DialContext)
			ips, err := r.LookupNetIP(context.TODO(), "ip4", "www.example.com")

			if err != nil {
				t.Error(err)
			} else {
				t.Logf("got %s", ips)
			}
		})
	}
}
