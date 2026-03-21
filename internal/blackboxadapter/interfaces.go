package blackboxadapter

import (
	"context"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

type ConfigLoader interface {
	Reload(path string) error
	Module(name string) (ModuleDef, bool)
	Modules() map[string]ModuleDef
}

type ProbeRunner interface {
	Run(ctx context.Context, module ModuleDef, target string, registry *prometheus.Registry, logger log.Logger) bool
}

type ProberRegistry interface {
	Register(name string, runner ProbeRunner)
	Get(name string) (ProbeRunner, bool)
}

type BackendAdapter interface {
	NewLoader() ConfigLoader
	DefaultRegistry() ProberRegistry
}
