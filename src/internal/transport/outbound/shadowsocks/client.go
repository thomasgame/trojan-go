package shadowsocks

import (
	"context"

	"github.com/thomasgame/trojan-go/common"
	"github.com/thomasgame/trojan-go/internal/core/config"
	"github.com/thomasgame/trojan-go/internal/core/tunnel"
	"github.com/thomasgame/trojan-go/internal/infra/log"
)

type Client struct {
	underlay tunnel.Client
	streamConnCipher
}

func (c *Client) DialConn(address *tunnel.Address, tunnel tunnel.Tunnel) (tunnel.Conn, error) {
	conn, err := c.underlay.DialConn(address, &Tunnel{})
	if err != nil {
		return nil, err
	}
	return &Conn{
		aeadConn: c.streamConnCipher.StreamConn(conn),
		Conn:     conn,
	}, nil
}

func (c *Client) DialPacket(tunnel tunnel.Tunnel) (tunnel.PacketConn, error) {
	panic("not supported")
}

func (c *Client) Close() error {
	return c.underlay.Close()
}

func NewClient(ctx context.Context, underlay tunnel.Client) (*Client, error) {
	cfg := config.FromContext(ctx, Name).(*Config)
	cipher, err := pickStreamCipher(cfg.Shadowsocks.Method, cfg.Shadowsocks.Password)
	if err != nil {
		return nil, common.NewError("invalid shadowsocks cipher").Base(err)
	}
	log.Debug("shadowsocks client created")
	return &Client{
		underlay:         underlay,
		streamConnCipher: cipher,
	}, nil
}
