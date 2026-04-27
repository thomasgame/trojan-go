package transport

import (
	"net"

	"github.com/thomasgame/trojan-go/internal/core/tunnel"
)

type Conn struct {
	net.Conn
}

func (c *Conn) Metadata() *tunnel.Metadata {
	return nil
}
