package blackboxadapter

import (
	bec "blackbox_agent/blackbox_exporter/config"
	"context"
	"testing"

	"github.com/go-kit/log"
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

	modules := loader.Modules()
	if len(modules) != 1 {
		t.Fatalf("unexpected modules length: %d", len(modules))
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
