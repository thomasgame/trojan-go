package mobilebind

import (
	"encoding/json"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

func TestProbeConfigJSONOverridesDialHostAndPreservesSNI(t *testing.T) {
	data, err := probeConfigJSON(`{
		"run_type": "server",
		"remote_addr": "vpn.example.com",
		"remote_port": 443,
		"ssl": {}
	}`, "203.0.113.10")
	if err != nil {
		t.Fatalf("probeConfigJSON returned error: %v", err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if got["run_type"] != "client" {
		t.Fatalf("run_type = %v, want client", got["run_type"])
	}
	if got["remote_addr"] != "203.0.113.10" {
		t.Fatalf("remote_addr = %v, want server ip", got["remote_addr"])
	}
	ssl := got["ssl"].(map[string]interface{})
	if ssl["sni"] != "vpn.example.com" {
		t.Fatalf("ssl.sni = %v, want original remote_addr", ssl["sni"])
	}
}

func TestProbeConfigJSONKeepsExplicitSNI(t *testing.T) {
	data, err := probeConfigJSON(`{
		"remote_addr": "vpn.example.com",
		"ssl": {"sni": "tls.example.com"}
	}`, "203.0.113.10")
	if err != nil {
		t.Fatalf("probeConfigJSON returned error: %v", err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	ssl := got["ssl"].(map[string]interface{})
	if ssl["sni"] != "tls.example.com" {
		t.Fatalf("ssl.sni = %v, want explicit sni", ssl["sni"])
	}
}

func TestProbeConfigJSONWithoutServerIPOnlyForcesClientMode(t *testing.T) {
	data, err := probeConfigJSON(`{
		"run_type": "server",
		"remote_addr": "vpn.example.com"
	}`, " ")
	if err != nil {
		t.Fatalf("probeConfigJSON returned error: %v", err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if got["run_type"] != "client" {
		t.Fatalf("run_type = %v, want client", got["run_type"])
	}
	if got["remote_addr"] != "vpn.example.com" {
		t.Fatalf("remote_addr = %v, want original host", got["remote_addr"])
	}
	if _, exists := got["ssl"]; exists {
		t.Fatalf("ssl unexpectedly created when serverIP is blank: %v", got["ssl"])
	}
}

func TestProbeConfigJSONRejectsInvalidJSON(t *testing.T) {
	if _, err := probeConfigJSON(`{`, "203.0.113.10"); err == nil {
		t.Fatal("probeConfigJSON accepted invalid json")
	}
}

func TestProbeRuntimeConfigJSONCanOverrideShadowsocks(t *testing.T) {
	disabled := false
	data, _, err := probeRuntimeConfigJSON(`{
		"remote_addr": "vpn.example.com",
		"shadowsocks": {"enabled": true, "method": "AES-128-GCM", "password": "secret"}
	}`, "203.0.113.10", &disabled)
	if err != nil {
		t.Fatalf("probeRuntimeConfigJSON returned error: %v", err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	shadowsocks := got["shadowsocks"].(map[string]interface{})
	if shadowsocks["enabled"] != false {
		t.Fatalf("shadowsocks.enabled = %v, want false", shadowsocks["enabled"])
	}
	if got["local_addr"] != "127.0.0.1" {
		t.Fatalf("local_addr = %v, want probe loopback", got["local_addr"])
	}
}

func TestProbeTargetHostPrefersServerIP(t *testing.T) {
	host, err := probeTargetHost(`{
		"remote_addr": " vpn.example.com "
	}`, " 203.0.113.10 ", "")
	if err != nil {
		t.Fatalf("probeTargetHost returned error: %v", err)
	}
	if host != "203.0.113.10" {
		t.Fatalf("probe target = %q, want server IP", host)
	}
}

func TestProbeTargetHostFallsBackToRemoteAddr(t *testing.T) {
	host, err := probeTargetHost(`{
		"remote_addr": " vpn.example.com "
	}`, " ", "")
	if err != nil {
		t.Fatalf("probeTargetHost returned error: %v", err)
	}
	if host != "vpn.example.com" {
		t.Fatalf("probe target = %q, want original remote_addr", host)
	}
}

func TestProbeTargetHostRejectsEmptyRemoteAddr(t *testing.T) {
	if _, err := probeTargetHost(`{"remote_addr": " "}`, " ", ""); err == nil {
		t.Fatal("probeTargetHost accepted empty remote_addr")
	}
}

func TestProbeTargetHostAllowsManualTargetOverride(t *testing.T) {
	host, err := probeTargetHost(`{"remote_addr": "vpn.example.com"}`, "203.0.113.10", "127.0.0.1")
	if err != nil {
		t.Fatalf("probeTargetHost returned error: %v", err)
	}
	if host != "127.0.0.1" {
		t.Fatalf("probe target = %q, want manual override", host)
	}
}

func TestProbeReadEOFIsUnreachableNotError(t *testing.T) {
	ok, err := probeReadResult(0, io.EOF)
	if err != nil {
		t.Fatalf("EOF should not be returned as probe error: %v", err)
	}
	if ok {
		t.Fatal("EOF should be classified as unreachable")
	}
}

func TestProbeUsesHTTPPort(t *testing.T) {
	if probePort != 80 {
		t.Fatalf("probePort = %d, want HTTP port 80", probePort)
	}
}

func TestManualHTTPProbeTarget34(t *testing.T) {
	if os.Getenv("FASTGO_MOBILEBIND_REAL_HTTP_PROBE") != "true" {
		t.Skip("set FASTGO_MOBILEBIND_REAL_HTTP_PROBE=true to probe the real nginx target")
	}
	const target = "34.96.140.96"
	conn, err := net.DialTimeout("tcp", target+":80", probeTimeout)
	if err != nil {
		t.Fatalf("dial %s:80 failed: %v", target, err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(probeTimeout)); err != nil {
		t.Fatalf("set deadline failed: %v", err)
	}
	req := "HEAD / HTTP/1.1\r\nHost: " + target + "\r\nConnection: close\r\n\r\n"
	if _, err := io.WriteString(conn, req); err != nil {
		t.Fatalf("write probe request failed: %v", err)
	}
	buf := make([]byte, 16)
	n, err := conn.Read(buf)
	ok, classifyErr := probeReadResult(n, err)
	if classifyErr != nil {
		t.Fatalf("probe read returned error: %v", classifyErr)
	}
	if !ok {
		t.Fatalf("probe target %s returned no readable response", target)
	}
}

func TestManualProbeWithClientConfigFile(t *testing.T) {
	configPath := strings.TrimSpace(os.Getenv("FASTGO_MOBILEBIND_REAL_CONFIG"))
	if configPath == "" {
		t.Skip("set FASTGO_MOBILEBIND_REAL_CONFIG to a Trojan-Go client JSON file")
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read client config failed: %v", err)
	}
	serverIP := os.Getenv("FASTGO_MOBILEBIND_REAL_SERVER_IP")
	targetHost := os.Getenv("FASTGO_MOBILEBIND_REAL_TARGET_HOST")
	ok, err := probeWithTargetHost(string(data), serverIP, targetHost)
	if err != nil {
		t.Fatalf("mobilebind Probe returned error: %v", err)
	}
	if !ok {
		t.Fatal("mobilebind Probe returned unreachable")
	}
}
