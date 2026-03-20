package blackboxadapter

import (
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log"
	bec "github.com/prometheus/blackbox_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/yaml.v3"
)

func TestNTPHelpers(t *testing.T) {
	host, port := splitNTPAddress("127.0.0.1:123")
	if host != "127.0.0.1" || port != "123" {
		t.Fatalf("unexpected host/port: %s %s", host, port)
	}

	host, port = splitNTPAddress(" time.google.com ")
	if host != "time.google.com" || port != ntpPort {
		t.Fatalf("unexpected host/port without port: %s %s", host, port)
	}

	if got := normalizeNTPServer("time.google.com", "123"); got != "time.google.com" {
		t.Fatalf("unexpected normalized server: %s", got)
	}
	if got := normalizeNTPServer("127.0.0.1", "124"); got != "127.0.0.1:124" {
		t.Fatalf("unexpected normalized server with custom port: %s", got)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	buf := make([]byte, 8)
	writeNTPTimestamp(buf, now)
	parsed := parseNTPTimestamp(buf)
	if diff := parsed.Sub(now); diff > time.Millisecond || diff < -time.Millisecond {
		t.Fatalf("unexpected parsed timestamp diff: %s", diff)
	}

	signed := make([]byte, 4)
	binary.BigEndian.PutUint32(signed, 0xffff0000)
	if got := parseShort(signed, true); got != -1 {
		t.Fatalf("unexpected signed short: %f", got)
	}

	unsigned := make([]byte, 4)
	binary.BigEndian.PutUint32(unsigned, 65536)
	if got := parseShort(unsigned, false); got != 1 {
		t.Fatalf("unexpected unsigned short: %f", got)
	}

	if got := parseReferenceID([]byte("GPS\x00"), 1, 4); got != "GPS" {
		t.Fatalf("unexpected stratum 1 reference id: %s", got)
	}
	if got := parseReferenceID([]byte{127, 0, 0, 1}, 2, 3); got != "127.0.0.1" {
		t.Fatalf("unexpected version 3 reference id: %s", got)
	}
	if got := parseReferenceID([]byte{0xde, 0xad, 0xbe, 0xef}, 2, 4); got != "deadbeef" {
		t.Fatalf("unexpected version 4 reference id: %s", got)
	}

	if ipHash(net.ParseIP("127.0.0.1")) == 0 {
		t.Fatal("expected non-zero hash")
	}
}

func TestChooseProtocolLiteralPaths(t *testing.T) {
	registry := prometheus.NewRegistry()
	ip, _, err := chooseProtocol(context.Background(), "ip4", false, "127.0.0.1", registry, newTestLogger())
	if err != nil {
		t.Fatal(err)
	}
	if ip.String() != "127.0.0.1" {
		t.Fatalf("unexpected ip: %s", ip.String())
	}

	registry = prometheus.NewRegistry()
	_, _, err = chooseProtocol(context.Background(), "ip6", false, "127.0.0.1", registry, newTestLogger())
	if err == nil || !strings.Contains(err.Error(), "no fallback") {
		t.Fatalf("expected no fallback error, got %v", err)
	}

	registry = prometheus.NewRegistry()
	ip, _, err = chooseProtocol(context.Background(), "ip6", true, "127.0.0.1", registry, newTestLogger())
	if err != nil {
		t.Fatal(err)
	}
	if ip.String() != "127.0.0.1" {
		t.Fatalf("unexpected fallback ip: %s", ip.String())
	}
}

func TestChooseProtocolLookupFailure(t *testing.T) {
	registry := prometheus.NewRegistry()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _, err := chooseProtocol(ctx, "ip4", false, "invalid.invalid", registry, newTestLogger())
	if err == nil {
		t.Fatal("expected lookup error")
	}
}

func TestQueryCollectProbeAndRunner(t *testing.T) {
	serverAddr := startMockNTPServer(t, 4)
	host, port, err := net.SplitHostPort(serverAddr)
	if err != nil {
		t.Fatal(err)
	}

	cfg := NTPProbeConfig{
		IPProtocol:          "ip4",
		IPProtocolFallback:  true,
		ProtocolVersion:     4,
		MeasurementDuration: 5 * time.Millisecond,
		HighDriftThreshold:  0,
	}

	ip := &net.IPAddr{IP: net.ParseIP(host)}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sample, err := queryNTP(ctx, ip, host, port, cfg, newTestLogger())
	if err != nil {
		t.Fatal(err)
	}
	if sample.Server != serverAddr {
		t.Fatalf("unexpected sample server: %s", sample.Server)
	}
	if sample.ReferenceID != "deadbeef" {
		t.Fatalf("unexpected reference id: %s", sample.ReferenceID)
	}

	registry := prometheus.NewRegistry()
	collected, err := collectNTPSample(ctx, serverAddr, cfg, registry, newTestLogger())
	if err != nil {
		t.Fatal(err)
	}
	if collected.Server != serverAddr {
		t.Fatalf("unexpected collected server: %s", collected.Server)
	}

	registry = prometheus.NewRegistry()
	if !probeNTP(ctx, serverAddr, cfg, registry, newTestLogger()) {
		t.Fatal("expected probeNTP success")
	}
	metrics, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	if len(metrics) == 0 {
		t.Fatal("expected gathered metrics")
	}

	runner := customNTPRunner{}
	registry = prometheus.NewRegistry()
	module := ModuleDef{Name: "ntp_probe", Prober: "ntp", NTP: cfg}
	if !runner.Run(ctx, module, serverAddr, registry, newTestLogger()) {
		t.Fatal("expected customNTPRunner success")
	}
}

func TestQueryAndProbeErrors(t *testing.T) {
	cfg := NTPProbeConfig{
		IPProtocol:         "ip4",
		IPProtocolFallback: true,
		SourceIPAddress:    "bad-ip",
		ProtocolVersion:    4,
	}
	ip := &net.IPAddr{IP: net.ParseIP("127.0.0.1")}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := queryNTP(ctx, ip, "127.0.0.1", "123", cfg, newTestLogger())
	if err == nil || !strings.Contains(err.Error(), "invalid source_ip_address") {
		t.Fatalf("expected invalid source_ip_address error, got %v", err)
	}

	cfg = NTPProbeConfig{
		IPProtocol:         "ip4",
		IPProtocolFallback: false,
		ProtocolVersion:    4,
	}
	if probeNTP(ctx, "not-a-real-host.invalid", cfg, prometheus.NewRegistry(), newTestLogger()) {
		t.Fatal("expected probeNTP failure")
	}
}

func TestQueryNTPUnexpectedMode(t *testing.T) {
	serverAddr := startMockNTPServer(t, 1)
	host, port, err := net.SplitHostPort(serverAddr)
	if err != nil {
		t.Fatal(err)
	}

	cfg := NTPProbeConfig{
		IPProtocol:         "ip4",
		IPProtocolFallback: true,
		ProtocolVersion:    4,
	}
	ip := &net.IPAddr{IP: net.ParseIP(host)}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = queryNTP(ctx, ip, host, port, cfg, newTestLogger())
	if err == nil || !strings.Contains(err.Error(), "unexpected NTP mode") {
		t.Fatalf("expected unexpected mode error, got %v", err)
	}
}

func TestTypesAndLoaderHelpers(t *testing.T) {
	var cfg NTPProbeConfig
	if err := yaml.Unmarshal([]byte("protocol_version: 5\n"), &cfg); err == nil {
		t.Fatal("expected protocol version validation error")
	}

	if err := yaml.Unmarshal([]byte("protocol_version: 4\nmeasurement_duration: -1s\n"), &cfg); err == nil {
		t.Fatal("expected negative measurement duration validation error")
	}

	if err := yaml.Unmarshal([]byte("protocol_version: 4\nhigh_drift_threshold: -1ms\n"), &cfg); err == nil {
		t.Fatal("expected negative high drift threshold validation error")
	}

	raw, ok := upstreamModuleFromDef(ModuleDef{Raw: bec.Module{Prober: "http"}})
	if !ok || raw.Prober != "http" {
		t.Fatal("expected upstream module conversion success")
	}
	if _, ok := upstreamModuleFromDef(ModuleDef{Raw: "bad"}); ok {
		t.Fatal("expected upstream module conversion failure")
	}

	safeConfig := bec.NewSafeConfig(prometheus.NewRegistry())
	loader := NewUpstreamConfigLoader(safeConfig, newTestLogger())
	if loader.SafeConfig() == nil {
		t.Fatal("expected non-nil safe config")
	}
	if loader.ntpConfig("missing").ProtocolVersion != defaultNTPProbeConfig.ProtocolVersion {
		t.Fatal("expected default ntp config")
	}

	file := writeTempFile(t, []byte("modules:\n  bad: ["))
	if _, err := loadNTPModuleConfigs(file); err == nil {
		t.Fatal("expected yaml parse error")
	}
	if _, _, err := sanitizeConfigForUpstream(file); err == nil {
		t.Fatal("expected sanitize yaml parse error")
	}
	if err := loader.Reload("/definitely/missing/file.yaml"); err == nil {
		t.Fatal("expected reload read error")
	}
	if _, _, err := sanitizeConfigForUpstream("/definitely/missing/file.yaml"); err == nil {
		t.Fatal("expected sanitize read error")
	}
}

func TestLoaderReloadAndUpstreamRunner(t *testing.T) {
	cfgFile := writeTempFile(t, []byte(`
modules:
  http_2xx:
    prober: http
    timeout: 5s
    http:
      method: GET
  ntp_probe:
    prober: ntp
    ntp:
      preferred_ip_protocol: ip4
      protocol_version: 4
`))

	backend := NewUpstreamBackend(prometheus.NewRegistry(), newTestLogger())
	loader := backend.NewLoader().(*UpstreamConfigLoader)
	if err := loader.Reload(cfgFile); err != nil {
		t.Fatal(err)
	}

	module, ok := loader.Module("ntp_probe")
	if !ok {
		t.Fatal("expected ntp probe module")
	}
	if module.NTP.IPProtocol != "ip4" {
		t.Fatalf("unexpected ntp preferred ip protocol: %s", module.NTP.IPProtocol)
	}

	modules := loader.Modules()
	if len(modules) != 2 {
		t.Fatalf("unexpected module count: %d", len(modules))
	}

	if _, ok := loader.Module("missing"); ok {
		t.Fatal("did not expect missing module")
	}

	called := false
	runner := upstreamProbeRunner{
		probeFn: func(_ context.Context, target string, module bec.Module, _ *prometheus.Registry, _ *slog.Logger) bool {
			called = true
			return target == "example.com" && module.Prober == "http"
		},
	}

	if !runner.Run(context.Background(), ModuleDef{Raw: bec.Module{Prober: "http"}}, "example.com", prometheus.NewRegistry(), newTestLogger()) {
		t.Fatal("expected upstream runner success")
	}
	if !called {
		t.Fatal("expected wrapped upstream probe fn to be called")
	}
	if runner.Run(context.Background(), ModuleDef{Raw: "bad"}, "example.com", prometheus.NewRegistry(), newTestLogger()) {
		t.Fatal("expected upstream runner failure with invalid raw module")
	}
	if NewUpstreamBackend(prometheus.NewRegistry(), newTestLogger()).NewLoader() == nil {
		t.Fatal("expected non-nil loader")
	}
	if runner, ok := backend.DefaultRegistry().Get("ntp"); !ok || runner == nil {
		t.Fatal("expected ntp runner in default registry")
	}
	if newSlogLogger() == nil || newDiscardSlogLogger() == nil {
		t.Fatal("expected slog logger helpers")
	}
}

func newTestLogger() log.Logger {
	return log.NewNopLogger()
}

func newDiscardSlogLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func writeTempFile(t *testing.T, content []byte) string {
	t.Helper()
	file, err := os.CreateTemp("", "adapter-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(content); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(file.Name()) })
	return file.Name()
}

func startMockNTPServer(t *testing.T, mode byte) string {
	t.Helper()
	conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })

	go func() {
		buf := make([]byte, ntpPacketSize)
		for {
			n, addr, err := conn.ReadFrom(buf)
			if err != nil {
				return
			}
			req := make([]byte, n)
			copy(req, buf[:n])
			resp := buildMockNTPResponse(req, mode)
			_, _ = conn.WriteTo(resp, addr)
		}
	}()

	return conn.LocalAddr().String()
}

func buildMockNTPResponse(req []byte, mode byte) []byte {
	resp := make([]byte, ntpPacketSize)
	resp[0] = mode
	resp[1] = 2
	resp[3] = 0xec

	rootDelay := make([]byte, 4)
	binary.BigEndian.PutUint32(rootDelay, 65536)
	copy(resp[4:8], rootDelay)

	rootDisp := make([]byte, 4)
	binary.BigEndian.PutUint32(rootDisp, 32768)
	copy(resp[8:12], rootDisp)

	binary.BigEndian.PutUint32(resp[12:16], 0xdeadbeef)

	reference := time.Now().UTC().Add(-time.Second)
	receive := time.Now().UTC().Add(-2 * time.Millisecond)
	transmit := time.Now().UTC().Add(-1 * time.Millisecond)

	writeNTPTimestamp(resp[16:24], reference)
	copy(resp[24:32], req[40:48])
	writeNTPTimestamp(resp[32:40], receive)
	writeNTPTimestamp(resp[40:48], transmit)
	return resp
}
