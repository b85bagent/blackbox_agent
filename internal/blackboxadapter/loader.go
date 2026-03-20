package blackboxadapter

import (
	"io"
	"log/slog"
	"os"

	bec "github.com/prometheus/blackbox_exporter/config"

	"github.com/go-kit/log"
	"gopkg.in/yaml.v3"
)

type UpstreamConfigLoader struct {
	safeConfig *bec.SafeConfig
	logger     log.Logger
	ntpModules map[string]NTPProbeConfig
}

func NewUpstreamConfigLoader(safeConfig *bec.SafeConfig, logger log.Logger) *UpstreamConfigLoader {
	return &UpstreamConfigLoader{
		safeConfig: safeConfig,
		logger:     logger,
	}
}

func (l *UpstreamConfigLoader) Reload(path string) error {
	ntpModules, err := loadNTPModuleConfigs(path)
	if err != nil {
		return err
	}

	sanitizedPath, cleanup, err := sanitizeConfigForUpstream(path)
	if err != nil {
		return err
	}
	defer cleanup()

	if err := l.safeConfig.ReloadConfig(sanitizedPath, newSlogLogger()); err != nil {
		return err
	}

	l.ntpModules = ntpModules
	return nil
}

func newSlogLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func (l *UpstreamConfigLoader) Module(name string) (ModuleDef, bool) {
	l.safeConfig.RLock()
	defer l.safeConfig.RUnlock()

	module, ok := l.safeConfig.C.Modules[name]
	if !ok {
		return ModuleDef{}, false
	}

	return moduleDefFromUpstream(name, module, l.ntpConfig(name)), true
}

func (l *UpstreamConfigLoader) Modules() map[string]ModuleDef {
	l.safeConfig.RLock()
	defer l.safeConfig.RUnlock()

	modules := make(map[string]ModuleDef, len(l.safeConfig.C.Modules))
	for name, module := range l.safeConfig.C.Modules {
		modules[name] = moduleDefFromUpstream(name, module, l.ntpConfig(name))
	}

	return modules
}

func (l *UpstreamConfigLoader) SafeConfig() *bec.SafeConfig {
	return l.safeConfig
}

func (l *UpstreamConfigLoader) ntpConfig(name string) NTPProbeConfig {
	if l.ntpModules == nil {
		return defaultNTPProbeConfig
	}

	if cfg, ok := l.ntpModules[name]; ok {
		return cfg
	}

	return defaultNTPProbeConfig
}

type adapterConfigFile struct {
	Modules map[string]adapterModuleConfig `yaml:"modules"`
}

type adapterModuleConfig struct {
	Prober string         `yaml:"prober,omitempty"`
	NTP    NTPProbeConfig `yaml:"ntp,omitempty"`
}

func loadNTPModuleConfigs(path string) (map[string]NTPProbeConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg adapterConfigFile
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, err
	}

	modules := make(map[string]NTPProbeConfig)
	for name, module := range cfg.Modules {
		if module.Prober != "ntp" {
			continue
		}
		modules[name] = module.NTP
	}

	return modules, nil
}

func sanitizeConfigForUpstream(path string) (string, func(), error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", func() {}, err
	}

	var node yaml.Node
	if err := yaml.Unmarshal(content, &node); err != nil {
		return "", func() {}, err
	}

	removeNTPFields(&node)

	sanitized, err := yaml.Marshal(&node)
	if err != nil {
		return "", func() {}, err
	}

	tmpFile, err := os.CreateTemp("", "blackbox-upstream-*.yaml")
	if err != nil {
		return "", func() {}, err
	}

	if _, err := tmpFile.Write(sanitized); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", func() {}, err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", func() {}, err
	}

	return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }, nil
}

func removeNTPFields(node *yaml.Node) {
	if node == nil {
		return
	}

	if node.Kind == yaml.MappingNode {
		filtered := make([]*yaml.Node, 0, len(node.Content))
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			value := node.Content[i+1]
			if key.Value == "ntp" {
				continue
			}
			removeNTPFields(value)
			filtered = append(filtered, key, value)
		}
		node.Content = filtered
		return
	}

	for _, child := range node.Content {
		removeNTPFields(child)
	}
}
