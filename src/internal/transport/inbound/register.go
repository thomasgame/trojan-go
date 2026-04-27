package inbound

import (
	_ "github.com/thomasgame/trojan-go/internal/transport/inbound/adapter"
	_ "github.com/thomasgame/trojan-go/internal/transport/inbound/dokodemo"
	_ "github.com/thomasgame/trojan-go/internal/transport/inbound/http"
	_ "github.com/thomasgame/trojan-go/internal/transport/inbound/socks"
	_ "github.com/thomasgame/trojan-go/internal/transport/inbound/tproxy"
)
