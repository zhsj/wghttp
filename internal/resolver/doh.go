package resolver

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var _ net.Conn = &dohConn{}

type dohConn struct {
	addr string

	once    sync.Once
	onceErr error

	in, ret bytes.Buffer
}

func (c *dohConn) Close() error                       { return nil }
func (c *dohConn) LocalAddr() net.Addr                { return nil }
func (c *dohConn) RemoteAddr() net.Addr               { return nil }
func (c *dohConn) SetDeadline(t time.Time) error      { return nil }
func (c *dohConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *dohConn) SetWriteDeadline(t time.Time) error { return nil }

func (c *dohConn) Write(b []byte) (int, error) { return c.in.Write(b) }

func (c *dohConn) Read(b []byte) (int, error) {
	c.once.Do(func() {
		url, err := url.Parse(c.addr)
		if err != nil {
			c.onceErr = err
			return
		}
		// RFC 8484
		url.Path = "/dns-query"

		// Skip 2 bytes which are length
		reqBody := bytes.NewReader(c.in.Bytes()[2:])
		req, err := http.NewRequest("POST", url.String(), reqBody)
		if err != nil {
			c.onceErr = err
			return
		}
		req.Header.Set("content-type", "application/dns-message")
		req.Header.Set("accept", "application/dns-message")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			c.onceErr = err
			return
		}
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			c.onceErr = err
			return
		}

		l := uint16(len(respBody))
		_, err = c.ret.Write([]byte{uint8(l >> 8), uint8(l & ((1 << 8) - 1))})
		if err != nil {
			c.onceErr = err
			return
		}

		_, err = c.ret.Write(respBody)
		if err != nil {
			c.onceErr = err
			return
		}
	})
	if c.onceErr != nil {
		return 0, c.onceErr
	}
	return c.ret.Read(b)
}
