//go:build nat || full || mini
// +build nat full mini

package build

import (
	_ "github.com/thomasgame/trojan-go/internal/app/wiring/modes/nat"
)
