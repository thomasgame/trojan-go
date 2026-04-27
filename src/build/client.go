//go:build client || full || mini
// +build client full mini

package build

import (
	_ "github.com/thomasgame/trojan-go/internal/app/wiring/modes/client"
)
