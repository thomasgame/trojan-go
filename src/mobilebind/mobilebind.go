package mobilebind

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/thomasgame/trojan-go/build"
	"github.com/thomasgame/trojan-go/common"
	"github.com/thomasgame/trojan-go/internal/core/proxy"
	"github.com/thomasgame/trojan-go/internal/infra/log"
)

const (
	probeTimeout = 8 * time.Second
	probePort    = 80
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

// Probe checks whether a Trojan-Go client JSON can reach the probe target.
func (*Mobilebind) Probe(configJSON string, serverIP string) (bool, error) {
	return Probe(configJSON, serverIP)
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

// Probe checks Trojan reachability through the same outbound stack used by client mode,
// without starting the long-lived local SOCKS runtime.
func Probe(configJSON string, serverIP string) (bool, error) {
	return probeWithTargetHost(configJSON, serverIP, "")
}

func probeWithTargetHost(configJSON string, serverIP string, targetHost string) (bool, error) {
	ok, err := probeWithTargetHostOnce(configJSON, serverIP, targetHost, nil)
	if err != nil || ok {
		return ok, err
	}
	if !probeShadowsocksEnabled(configJSON) {
		return false, nil
	}
	disabled := false
	log.Warn("mobilebind probe got no response with shadowsocks enabled, retrying base trojan reachability")
	return probeWithTargetHostOnce(configJSON, serverIP, targetHost, &disabled)
}

func probeWithTargetHostOnce(configJSON string, serverIP string, targetHost string, shadowsocksEnabled *bool) (bool, error) {
	probeHost, err := probeTargetHost(configJSON, serverIP, targetHost)
	if err != nil {
		return false, err
	}
	data, listenAddr, err := probeRuntimeConfigJSON(configJSON, serverIP, shadowsocksEnabled)
	if err != nil {
		return false, err
	}
	clientProxy, err := proxy.NewProxyFromConfigData(data, true)
	if err != nil {
		return false, err
	}
	runErr := make(chan error, 1)
	go func() {
		runErr <- clientProxy.Run()
	}()
	defer clientProxy.Close()
	if err := waitProbeProxyReady(listenAddr, runErr); err != nil {
		return false, err
	}
	conn, err := dialProbeTargetViaSocks(listenAddr, probeHost, probePort)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	deadline := time.Now().Add(probeTimeout)
	if err := conn.SetDeadline(deadline); err != nil {
		return false, err
	}
	req := "HEAD / HTTP/1.1\r\nHost: " + probeHost + "\r\nConnection: close\r\n\r\n"
	if _, err := io.WriteString(conn, req); err != nil {
		return false, err
	}
	buf := make([]byte, 16)
	n, err := conn.Read(buf)
	return probeReadResult(n, err)
}

func probeShadowsocksEnabled(configJSON string) bool {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &raw); err != nil {
		return false
	}
	shadowsocks, _ := raw["shadowsocks"].(map[string]interface{})
	enabled, _ := shadowsocks["enabled"].(bool)
	return enabled
}

func dialProbeTargetViaSocks(proxyAddr string, targetHost string, targetPort int) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", proxyAddr, probeTimeout)
	if err != nil {
		return nil, err
	}
	if err := conn.SetDeadline(time.Now().Add(probeTimeout)); err != nil {
		conn.Close()
		return nil, err
	}
	if _, err := conn.Write([]byte{0x05, 0x01, 0x00}); err != nil {
		conn.Close()
		return nil, err
	}
	resp := [2]byte{}
	if _, err := io.ReadFull(conn, resp[:]); err != nil {
		conn.Close()
		return nil, err
	}
	if resp != [2]byte{0x05, 0x00} {
		conn.Close()
		return nil, common.NewError("probe socks handshake rejected")
	}
	cleanTargetHost := strings.TrimSpace(targetHost)
	if cleanTargetHost == "" {
		conn.Close()
		return nil, common.NewError("probe target host length is invalid")
	}
	req := []byte{0x05, 0x01, 0x00}
	if ip := net.ParseIP(cleanTargetHost); ip != nil {
		if ipv4 := ip.To4(); ipv4 != nil {
			req = append(req, 0x01)
			req = append(req, ipv4...)
		} else {
			req = append(req, 0x04)
			req = append(req, ip.To16()...)
		}
	} else {
		hostBytes := []byte(cleanTargetHost)
		if len(hostBytes) > 255 {
			conn.Close()
			return nil, common.NewError("probe target host length is invalid")
		}
		req = append(req, 0x03, byte(len(hostBytes)))
		req = append(req, hostBytes...)
	}
	port := make([]byte, 2)
	binary.BigEndian.PutUint16(port, uint16(targetPort))
	req = append(req, port...)
	if _, err := conn.Write(req); err != nil {
		conn.Close()
		return nil, err
	}
	header := [4]byte{}
	if _, err := io.ReadFull(conn, header[:]); err != nil {
		conn.Close()
		return nil, err
	}
	if header[0] != 0x05 || header[1] != 0x00 {
		conn.Close()
		return nil, common.NewError("probe socks connect failed")
	}
	if err := discardSocksBindAddress(conn, header[3]); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func discardSocksBindAddress(conn net.Conn, atyp byte) error {
	switch atyp {
	case 0x01:
		_, err := io.CopyN(io.Discard, conn, 4+2)
		return err
	case 0x03:
		length := [1]byte{}
		if _, err := io.ReadFull(conn, length[:]); err != nil {
			return err
		}
		_, err := io.CopyN(io.Discard, conn, int64(length[0])+2)
		return err
	case 0x04:
		_, err := io.CopyN(io.Discard, conn, 16+2)
		return err
	default:
		return common.NewError("probe socks reply address type is invalid")
	}
}

func waitProbeProxyReady(listenAddr string, runErr <-chan error) error {
	deadline := time.Now().Add(probeTimeout)
	for time.Now().Before(deadline) {
		select {
		case err := <-runErr:
			if err != nil {
				return common.NewError("probe client proxy stopped before listening").Base(err)
			}
			return common.NewError("probe client proxy stopped before listening")
		default:
		}
		conn, err := net.DialTimeout("tcp", listenAddr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return common.NewError("probe client proxy listen timeout: " + listenAddr)
}

func probeReadResult(n int, err error) (bool, error) {
	if err != nil {
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			return false, nil
		}
		return false, err
	}
	return n > 0, nil
}

func probeConfigJSON(configJSON string, serverIP string) ([]byte, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &raw); err != nil {
		return nil, common.NewError("failed to parse probe config").Base(err)
	}
	raw["run_type"] = "client"
	cleanServerIP := strings.TrimSpace(serverIP)
	remoteHost, _ := raw["remote_addr"].(string)
	cleanRemoteHost := strings.TrimSpace(remoteHost)
	if cleanServerIP != "" && cleanRemoteHost != "" {
		raw["remote_addr"] = cleanServerIP
		ssl, _ := raw["ssl"].(map[string]interface{})
		if ssl == nil {
			ssl = make(map[string]interface{})
			raw["ssl"] = ssl
		}
		sni, _ := ssl["sni"].(string)
		if strings.TrimSpace(sni) == "" {
			ssl["sni"] = cleanRemoteHost
		}
	}
	return json.Marshal(raw)
}

func probeRuntimeConfigJSON(configJSON string, serverIP string, shadowsocksEnabled *bool) ([]byte, string, error) {
	data, err := probeConfigJSON(configJSON, serverIP)
	if err != nil {
		return nil, "", err
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, "", common.NewError("failed to parse normalized probe config").Base(err)
	}
	const localHost = "127.0.0.1"
	localPort := common.PickPort("tcp", localHost)
	raw["local_addr"] = localHost
	raw["local_port"] = localPort
	if shadowsocksEnabled != nil {
		shadowsocks, _ := raw["shadowsocks"].(map[string]interface{})
		if shadowsocks == nil {
			shadowsocks = make(map[string]interface{})
			raw["shadowsocks"] = shadowsocks
		}
		shadowsocks["enabled"] = *shadowsocksEnabled
	}
	next, err := json.Marshal(raw)
	if err != nil {
		return nil, "", err
	}
	return next, net.JoinHostPort(localHost, strconv.Itoa(localPort)), nil
}

func probeTargetHost(configJSON string, serverIP string, targetHost string) (string, error) {
	cleanTargetHost := strings.TrimSpace(targetHost)
	if cleanTargetHost != "" {
		return cleanTargetHost, nil
	}
	cleanServerIP := strings.TrimSpace(serverIP)
	if cleanServerIP != "" {
		return cleanServerIP, nil
	}
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &raw); err != nil {
		return "", common.NewError("failed to parse probe config").Base(err)
	}
	remoteHost, _ := raw["remote_addr"].(string)
	cleanRemoteHost := strings.TrimSpace(remoteHost)
	if cleanRemoteHost == "" {
		return "", common.NewError("probe target remote_addr is empty")
	}
	return cleanRemoteHost, nil
}
