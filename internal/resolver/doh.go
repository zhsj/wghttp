package resolver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

var _ net.Conn = &dohConn{}

type dohConn struct {
	query, resp *bytes.Buffer

	do func() error
}

func newDoHConn(ctx context.Context, client *http.Client, addr string) (*dohConn, error) {
	c := &dohConn{
		query: &bytes.Buffer{},
		resp:  &bytes.Buffer{},
	}

	url, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	// RFC 8484
	url.Path = "/dns-query"

	c.do = func() error {
		if c.query.Len() <= 2 || c.resp.Len() > 0 {
			return nil
		}

		// Skip length header
		c.query.Next(2)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url.String(), c.query)
		if err != nil {
			return err
		}
		req.Header.Set("content-type", "application/dns-message")
		req.Header.Set("accept", "application/dns-message")

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server return %d: %s", resp.StatusCode, respBody)
		}

		// Add length header
		l := uint16(len(respBody))
		_, err = c.resp.Write([]byte{uint8(l >> 8), uint8(l & ((1 << 8) - 1))})
		if err != nil {
			return err
		}

		_, err = c.resp.Write(respBody)
		return err
	}

	return c, nil
}

func (c *dohConn) Close() error                     { return nil }
func (c *dohConn) LocalAddr() net.Addr              { return nil }
func (c *dohConn) RemoteAddr() net.Addr             { return nil }
func (c *dohConn) SetDeadline(time.Time) error      { return nil }
func (c *dohConn) SetReadDeadline(time.Time) error  { return nil }
func (c *dohConn) SetWriteDeadline(time.Time) error { return nil }

func (c *dohConn) Write(b []byte) (int, error) { return c.query.Write(b) }

func (c *dohConn) Read(b []byte) (int, error) {
	if err := c.do(); err != nil {
		return 0, err
	}

	return c.resp.Read(b)
}
