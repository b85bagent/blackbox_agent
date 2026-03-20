package module

import (
	"errors"
	"time"
)

func validateNTPConfig(ntpConfigCheck map[string]interface{}, moduleName string) (bool, error) {
	ntpConfig, ok := ntpConfigCheck["ntp"].(map[interface{}]interface{})
	if !ok {
		return false, errors.New("NTP_prober_name error in [" + moduleName + "]")
	}

	if value, ok := ntpConfig["preferred_ip_protocol"]; ok {
		if _, ok := value.(string); !ok {
			return false, errors.New("NTP_preferred_ip_protocol must be a string in [" + moduleName + "]")
		}
	}

	if value, ok := ntpConfig["ip_protocol_fallback"]; ok {
		if _, ok := value.(bool); !ok {
			return false, errors.New("NTP_ip_protocol_fallback must be a bool in [" + moduleName + "]")
		}
	}

	if value, ok := ntpConfig["source_ip_address"]; ok {
		if _, ok := value.(string); !ok {
			return false, errors.New("NTP_source_ip_address must be a string in [" + moduleName + "]")
		}
	}

	if value, ok := ntpConfig["protocol_version"]; ok {
		if protocolVersion, ok := value.(int); !ok || protocolVersion < 2 || protocolVersion > 4 {
			return false, errors.New("NTP_protocol_version must be 2, 3 or 4 in [" + moduleName + "]")
		}
	}

	if value, ok := ntpConfig["measurement_duration"]; ok {
		duration, ok := value.(string)
		if !ok {
			return false, errors.New("NTP_measurement_duration must be a duration string in [" + moduleName + "]")
		}
		if _, err := time.ParseDuration(duration); err != nil {
			return false, errors.New("NTP_measurement_duration format invalid in [" + moduleName + "]")
		}
	}

	if value, ok := ntpConfig["high_drift_threshold"]; ok {
		duration, ok := value.(string)
		if !ok {
			return false, errors.New("NTP_high_drift_threshold must be a duration string in [" + moduleName + "]")
		}
		if _, err := time.ParseDuration(duration); err != nil {
			return false, errors.New("NTP_high_drift_threshold format invalid in [" + moduleName + "]")
		}
	}

	return true, nil
}
