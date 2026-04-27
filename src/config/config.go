package config

import (
	"context"

	coreconfig "github.com/thomasgame/trojan-go/internal/core/config"
)

type Creator = coreconfig.Creator

var RegisterConfigCreator = coreconfig.RegisterConfigCreator
var WithJSONConfig = coreconfig.WithJSONConfig
var WithYAMLConfig = coreconfig.WithYAMLConfig
var WithConfig = coreconfig.WithConfig

func FromContext(ctx context.Context, name string) interface{} {
	return coreconfig.FromContext(ctx, name)
}
