package main

import (
	"encoding/base64"

	"golang.zx2c4.com/wireguard/conn"
)

type connBind struct {
	// magic 3 bytes in wireguard header reserved section.
	clientID    []uint8
	defaultBind conn.Bind
}

func newConnBind(clientID string) conn.Bind {
	defaultBind := conn.NewDefaultBind()
	if clientID == "" {
		return defaultBind
	}
	parsed, err := base64.StdEncoding.DecodeString(clientID)
	if err != nil {
		logger.Errorf("Invalid client id: %v, fallback to default", err)
		return defaultBind
	}
	return &connBind{clientID: parsed, defaultBind: defaultBind}
}

func (c *connBind) Open(port uint16) ([]conn.ReceiveFunc, uint16, error) {
	fns, actualPort, err := c.defaultBind.Open(port)
	newFNs := make([]conn.ReceiveFunc, 0, len(fns))
	for i := range fns {
		f := fns[i]
		newFNs = append(newFNs, func(packets [][]byte, sizes []int, eps []conn.Endpoint) (n int, err error) {
			n, err = f(packets, sizes, eps)
			for i := range packets {
				if len(packets[i]) > 4 {
					copy(packets[i][1:4], []byte{0, 0, 0})
				}
			}
			return
		})
	}
	return newFNs, actualPort, err
}

func (c *connBind) BatchSize() int {
	return c.defaultBind.BatchSize()
}

func (c *connBind) Close() error { return c.defaultBind.Close() }

func (c *connBind) SetMark(mark uint32) error { return c.defaultBind.SetMark(mark) }

func (c *connBind) Send(bufs [][]byte, ep conn.Endpoint) error {
	for i := range bufs {
		if len(bufs[i]) > 4 {
			copy(bufs[i][1:4], c.clientID)
		}
	}
	return c.defaultBind.Send(bufs, ep)
}

func (c *connBind) ParseEndpoint(s string) (conn.Endpoint, error) {
	return c.defaultBind.ParseEndpoint(s)
}
