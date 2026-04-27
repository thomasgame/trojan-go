//go:build server || full || mini
// +build server full mini

package build

import (
	_ "github.com/thomasgame/trojan-go/internal/app/wiring/modes/server"
)
