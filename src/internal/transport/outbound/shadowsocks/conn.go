package shadowsocks

import (
	"net"

	"github.com/thomasgame/trojan-go/internal/core/tunnel"
)

type Conn struct {
	aeadConn net.Conn
	tunnel.Conn
	prefix []byte
}

func (c *Conn) Read(p []byte) (n int, err error) {
	if len(c.prefix) > 0 {
		n = copy(p, c.prefix)
		c.prefix = c.prefix[n:]
		if n == len(p) {
			return n, nil
		}
		m, err := c.aeadConn.Read(p[n:])
		return n + m, err
	}
	return c.aeadConn.Read(p)
}

func (c *Conn) Write(p []byte) (n int, err error) {
	return c.aeadConn.Write(p)
}

func (c *Conn) Close() error {
	c.Conn.Close()
	return c.aeadConn.Close()
}

func (c *Conn) Metadata() *tunnel.Metadata {
	return c.Conn.Metadata()
}
