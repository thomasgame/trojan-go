package mobilebind

import (
	"os"
	"sync"

	_ "github.com/thomasgame/trojan-go/build"
	"github.com/thomasgame/trojan-go/common"
	"github.com/thomasgame/trojan-go/internal/core/proxy"
	"github.com/thomasgame/trojan-go/internal/infra/log"
)

var (
	runtimeMu sync.Mutex
	runtime   *engineRuntime
)

type engineRuntime struct {
	proxy   *proxy.Proxy
	running bool
}

// Mobilebind is a gomobile-friendly facade for the package-level runtime helpers.
//
// Keeping this exported type allows gobind to see a bindable symbol even on
// toolchains where package-only exported functions are not discovered reliably.
type Mobilebind struct{}

// NewMobilebind creates a zero-state facade that forwards to the package runtime.
func NewMobilebind() *Mobilebind {
	return &Mobilebind{}
}

// Start loads a Trojan-Go JSON config file and starts one local SOCKS client runtime.
func (*Mobilebind) Start(configPath string) error {
	return Start(configPath)
}

// Stop terminates the currently running Trojan-Go client runtime, if any.
func (*Mobilebind) Stop() error {
	return Stop()
}

// IsRunning reports whether the local Trojan-Go client runtime is still serving the SOCKS port.
func (*Mobilebind) IsRunning() bool {
	return IsRunning()
}

// Start loads a Trojan-Go JSON config file and starts one local SOCKS client runtime.
//
// The Android side only relies on the local SOCKS listener becoming reachable; tun2socks
// remains outside of this mobilebind package.
func Start(configPath string) error {
	runtimeMu.Lock()
	defer runtimeMu.Unlock()

	if runtime != nil && runtime.running {
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return common.NewError("failed to read config file").Base(err)
	}

	client, err := proxy.NewProxyFromConfigData(data, true)
	if err != nil {
		return common.NewError("failed to create proxy from config").Base(err)
	}

	next := &engineRuntime{
		proxy:   client,
		running: true,
	}
	runtime = next

	go func(holder *engineRuntime) {
		err := holder.proxy.Run()
		if err != nil {
			log.Error(common.NewError("mobilebind proxy runtime stopped").Base(err))
		}
		runtimeMu.Lock()
		if runtime == holder {
			runtime.running = false
		}
		runtimeMu.Unlock()
	}(next)

	return nil
}

// Stop terminates the currently running Trojan-Go client runtime, if any.
func Stop() error {
	runtimeMu.Lock()
	defer runtimeMu.Unlock()

	if runtime == nil {
		return nil
	}
	if runtime.proxy != nil {
		if err := runtime.proxy.Close(); err != nil {
			return common.NewError("failed to stop proxy runtime").Base(err)
		}
	}
	runtime.running = false
	runtime = nil
	return nil
}

// IsRunning reports whether the local Trojan-Go client runtime is still serving the SOCKS port.
func IsRunning() bool {
	runtimeMu.Lock()
	defer runtimeMu.Unlock()

	return runtime != nil && runtime.running
}
