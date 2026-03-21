package blackboxadapter

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/go-kit/log"
	bec "github.com/prometheus/blackbox_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
)

type stubProbeRunner struct {
	called bool
}

func (s *stubProbeRunner) Run(_ context.Context, _ ModuleDef, _ string, _ *prometheus.Registry, _ log.Logger) bool {
	s.called = true
	return true
}

func TestRegistryRegisterAndGet(t *testing.T) {
	registry := NewRegistry()
	runner := &stubProbeRunner{}

	registry.Register("http", runner)

	got, ok := registry.Get("http")
	if !ok {
		t.Fatal("expected registered runner")
	}

	if got != runner {
		t.Fatal("expected same runner instance")
	}
}

func TestUpstreamConfigLoaderModuleAndModules(t *testing.T) {
	safeConfig := bec.NewSafeConfig(prometheus.NewRegistry())
	safeConfig.C.Modules = map[string]bec.Module{}
	safeConfig.C.Modules["http_2xx"] = bec.Module{Prober: "http"}

	loader := NewUpstreamConfigLoader(safeConfig, log.NewNopLogger())

	module, ok := loader.Module("http_2xx")
	if !ok {
		t.Fatal("expected module to exist")
	}

	if module.Name != "http_2xx" {
		t.Fatalf("unexpected module name: %s", module.Name)
	}

	if module.Prober != "http" {
		t.Fatalf("unexpected prober: %s", module.Prober)
	}

	if module.NTP.ProtocolVersion != defaultNTPProbeConfig.ProtocolVersion {
		t.Fatalf("unexpected default ntp protocol version: %d", module.NTP.ProtocolVersion)
	}

	modules := loader.Modules()
	if len(modules) != 1 {
		t.Fatalf("unexpected modules length: %d", len(modules))
	}
}

func TestLoadNTPModuleConfigs(t *testing.T) {
	content := []byte(`
modules:
  ntp_probe:
    prober: ntp
    ntp:
      preferred_ip_protocol: ip4
      ip_protocol_fallback: false
      source_ip_address: 127.0.0.1
      protocol_version: 3
      measurement_duration: 5s
      high_drift_threshold: 25ms
  http_2xx:
    prober: http
`)

	file, err := os.CreateTemp("", "ntp-config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	if _, err := file.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	modules, err := loadNTPModuleConfigs(file.Name())
	if err != nil {
		t.Fatal(err)
	}

	cfg, ok := modules["ntp_probe"]
	if !ok {
		t.Fatal("expected ntp module config")
	}

	if cfg.ProtocolVersion != 3 {
		t.Fatalf("unexpected protocol version: %d", cfg.ProtocolVersion)
	}
	if cfg.IPProtocol != "ip4" {
		t.Fatalf("unexpected preferred ip protocol: %s", cfg.IPProtocol)
	}
	if cfg.IPProtocolFallback {
		t.Fatal("expected fallback to be false")
	}
	if _, ok := modules["http_2xx"]; ok {
		t.Fatal("did not expect non-ntp module config")
	}
}

func TestSanitizeConfigForUpstreamRemovesNTPField(t *testing.T) {
	content := []byte(`
modules:
  ntp_probe:
    prober: ntp
    ntp:
      preferred_ip_protocol: ip4
  http_2xx:
    prober: http
    http:
      method: GET
`)

	file, err := os.CreateTemp("", "sanitize-config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	if _, err := file.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	sanitizedPath, cleanup, err := sanitizeConfigForUpstream(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	sanitized, err := os.ReadFile(sanitizedPath)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(sanitized), "ntp:") {
		t.Fatal("expected sanitized config to remove ntp field")
	}
	if !strings.Contains(string(sanitized), "method: GET") {
		t.Fatal("expected sanitized config to keep non-ntp fields")
	}
}

func TestUpstreamBackendDefaultRegistry(t *testing.T) {
	backend := NewUpstreamBackend(prometheus.NewRegistry(), log.NewNopLogger())
	registry := backend.DefaultRegistry()

	for _, name := range []string{"http", "tcp", "icmp", "dns", "grpc", "ntp"} {
		if _, ok := registry.Get(name); !ok {
			t.Fatalf("expected runner for %s", name)
		}
	}
}

func TestDefaultRegistryUsesCustomNTPRunner(t *testing.T) {
	backend := NewUpstreamBackend(prometheus.NewRegistry(), log.NewNopLogger())
	registry := backend.DefaultRegistry()

	runner, ok := registry.Get("ntp")
	if !ok {
		t.Fatal("expected ntp runner")
	}

	if _, ok := runner.(customNTPRunner); !ok {
		t.Fatalf("expected customNTPRunner, got %T", runner)
	}
}
