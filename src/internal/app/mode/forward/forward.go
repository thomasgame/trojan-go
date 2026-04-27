package forward

import (
	"context"

	"github.com/thomasgame/trojan-go/internal/app/mode/client"
	"github.com/thomasgame/trojan-go/internal/core/config"
	"github.com/thomasgame/trojan-go/internal/core/proxy"
	"github.com/thomasgame/trojan-go/internal/core/tunnel"
	"github.com/thomasgame/trojan-go/internal/transport/inbound/dokodemo"
)

const Name = "FORWARD"

func init() {
	proxy.RegisterProxyCreator(Name, func(ctx context.Context) (*proxy.Proxy, error) {
		cfg := config.FromContext(ctx, Name).(*client.Config)
		ctx, cancel := context.WithCancel(ctx)
		serverStack := []string{dokodemo.Name}
		clientStack := client.GenerateClientTree(cfg.TransportPlugin.Enabled, cfg.Mux.Enabled, cfg.Websocket.Enabled, cfg.Shadowsocks.Enabled, cfg.Router.Enabled)
		c, err := proxy.CreateClientStack(ctx, clientStack)
		if err != nil {
			cancel()
			return nil, err
		}
		s, err := proxy.CreateServerStack(ctx, serverStack)
		if err != nil {
			cancel()
			return nil, err
		}
		return proxy.NewProxy(ctx, cancel, []tunnel.Server{s}, c), nil
	})
}

func init() {
	config.RegisterConfigCreator(Name, func() interface{} {
		return new(client.Config)
	})
}
