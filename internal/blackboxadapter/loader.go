package blackboxadapter

import (
	bec "blackbox_agent/blackbox_exporter/config"

	"github.com/go-kit/log"
)

type UpstreamConfigLoader struct {
	safeConfig *bec.SafeConfig
	logger     log.Logger
}

func NewUpstreamConfigLoader(safeConfig *bec.SafeConfig, logger log.Logger) *UpstreamConfigLoader {
	return &UpstreamConfigLoader{
		safeConfig: safeConfig,
		logger:     logger,
	}
}

func (l *UpstreamConfigLoader) Reload(path string) error {
	return l.safeConfig.ReloadConfig(path, l.logger)
}

func (l *UpstreamConfigLoader) Module(name string) (ModuleDef, bool) {
	l.safeConfig.RLock()
	defer l.safeConfig.RUnlock()

	module, ok := l.safeConfig.C.Modules[name]
	if !ok {
		return ModuleDef{}, false
	}

	return moduleDefFromUpstream(name, module), true
}

func (l *UpstreamConfigLoader) Modules() map[string]ModuleDef {
	l.safeConfig.RLock()
	defer l.safeConfig.RUnlock()

	modules := make(map[string]ModuleDef, len(l.safeConfig.C.Modules))
	for name, module := range l.safeConfig.C.Modules {
		modules[name] = moduleDefFromUpstream(name, module)
	}

	return modules
}

func (l *UpstreamConfigLoader) SafeConfig() *bec.SafeConfig {
	return l.safeConfig
}
