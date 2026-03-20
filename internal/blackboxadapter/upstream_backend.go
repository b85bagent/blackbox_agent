package blackboxadapter

import (
	bec "blackbox_agent/blackbox_exporter/config"
	bep "blackbox_agent/blackbox_exporter/prober"
	"context"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

type UpstreamBackend struct {
	registerer prometheus.Registerer
	logger     log.Logger
}

func NewUpstreamBackend(registerer prometheus.Registerer, logger log.Logger) *UpstreamBackend {
	return &UpstreamBackend{
		registerer: registerer,
		logger:     logger,
	}
}

func (b *UpstreamBackend) NewLoader() ConfigLoader {
	return NewUpstreamConfigLoader(bec.NewSafeConfig(b.registerer), b.logger)
}

func (b *UpstreamBackend) DefaultRegistry() ProberRegistry {
	registry := NewRegistry()
	registry.Register("http", upstreamProbeRunner{probeFn: bep.ProbeHTTP})
	registry.Register("tcp", upstreamProbeRunner{probeFn: bep.ProbeTCP})
	registry.Register("icmp", upstreamProbeRunner{probeFn: bep.ProbeICMP})
	registry.Register("dns", upstreamProbeRunner{probeFn: bep.ProbeDNS})
	registry.Register("grpc", upstreamProbeRunner{probeFn: bep.ProbeGRPC})
	registry.Register("ntp", customNTPRunner{})

	return registry
}

type upstreamProbeRunner struct {
	probeFn bep.ProbeFn
}

func (r upstreamProbeRunner) Run(ctx context.Context, module ModuleDef, target string, registry *prometheus.Registry, logger log.Logger) bool {
	upstreamModule, ok := upstreamModuleFromDef(module)
	if !ok {
		return false
	}

	return r.probeFn(ctx, target, upstreamModule, registry, logger)
}
