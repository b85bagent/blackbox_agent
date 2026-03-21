package blackboxadapter

import (
	"errors"
	"time"

	bec "github.com/prometheus/blackbox_exporter/config"
)

// ModuleDef is the stable module shape exposed to business code.
// Raw keeps the backend-specific module during the migration period.
type ModuleDef struct {
	Name   string
	Prober string
	NTP    NTPProbeConfig
	Raw    any
}

type NTPProbeConfig struct {
	IPProtocol          string        `yaml:"preferred_ip_protocol,omitempty"`
	IPProtocolFallback  bool          `yaml:"ip_protocol_fallback,omitempty"`
	SourceIPAddress     string        `yaml:"source_ip_address,omitempty"`
	ProtocolVersion     int           `yaml:"protocol_version,omitempty"`
	MeasurementDuration time.Duration `yaml:"measurement_duration,omitempty"`
	HighDriftThreshold  time.Duration `yaml:"high_drift_threshold,omitempty"`
}

var defaultNTPProbeConfig = NTPProbeConfig{
	ProtocolVersion:     4,
	IPProtocolFallback:  true,
	MeasurementDuration: 30 * time.Second,
	HighDriftThreshold:  10 * time.Millisecond,
}

func moduleDefFromUpstream(name string, module bec.Module, ntpConfig NTPProbeConfig) ModuleDef {
	return ModuleDef{
		Name:   name,
		Prober: module.Prober,
		NTP:    ntpConfig,
		Raw:    module,
	}
}

func upstreamModuleFromDef(module ModuleDef) (bec.Module, bool) {
	rawModule, ok := module.Raw.(bec.Module)
	if !ok {
		return bec.Module{}, false
	}

	return rawModule, true
}

func (c *NTPProbeConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = defaultNTPProbeConfig
	type plain NTPProbeConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}

	if c.ProtocolVersion < 2 || c.ProtocolVersion > 4 {
		return errors.New("protocol_version must be one of 2, 3 or 4")
	}
	if c.MeasurementDuration < 0 {
		return errors.New("measurement_duration cannot be negative")
	}
	if c.HighDriftThreshold < 0 {
		return errors.New("high_drift_threshold cannot be negative")
	}

	return nil
}
