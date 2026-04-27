package server

import (
	"github.com/thomasgame/trojan-go/internal/app/mode/client"
	"github.com/thomasgame/trojan-go/internal/core/config"
)

func init() {
	config.RegisterConfigCreator(Name, func() interface{} {
		return new(client.Config)
	})
}
