//go:build forward || full || mini
// +build forward full mini

package build

import (
	_ "github.com/thomasgame/trojan-go/internal/app/wiring/modes/forward"
)
