package bootstrap

import (
	"flag"
	"sync"

	"github.com/thomasgame/trojan-go/internal/app/runtime/options"
	"github.com/thomasgame/trojan-go/internal/infra/log"
)

var parseFlagsOnce sync.Once

// Run starts Trojan-Go using the registered option handlers.
func Run() {
	parseFlagsOnce.Do(flag.Parse)
	for {
		h, err := option.PopOptionHandler()
		if err != nil {
			log.Fatal("invalid options")
		}
		if err := h.Handle(); err == nil {
			return
		}
	}
}
