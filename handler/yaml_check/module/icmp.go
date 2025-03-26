package module

import (
	"errors"
	"log"
)

func validateICMPConfig(icmpConfigCheck map[string]interface{}, moduleName string) (bool, error) {

	icmpConfig, ok := icmpConfigCheck["icmp"].(map[interface{}]interface{})
	if !ok {
		log.Println(icmpConfigCheck)
		return false, errors.New("ICMP_prober_name error in [" + moduleName + "]")
	}

	// 验证 preferred_ip_protocol 是 string
	if value, ok := icmpConfig["preferred_ip_protocol"]; ok {
		if _, ok := value.(string); !ok {
			return false, errors.New("ICMP_preferred_ip_protocol must be a string in [" + moduleName + "]")
		}
	}

	// 验证 ip_protocol_fallback 是 bool
	if value, ok := icmpConfig["ip_protocol_fallback"]; ok {
		if _, ok := value.(bool); !ok {
			return false, errors.New("ICMP_ip_protocol_fallback must be a bool in [" + moduleName + "]")
		}
	}

	// 验证 source_ip_address 是 string
	if value, ok := icmpConfig["source_ip_address"]; ok {
		if _, ok := value.(string); !ok {
			return false, errors.New("ICMP_source_ip_address must be a string in [" + moduleName + "]")
		}
	}

	// 验证 dont_fragment 是 bool
	if value, ok := icmpConfig["dont_fragment"]; ok {
		if _, ok := value.(bool); !ok {
			return false, errors.New("ICMP_dont_fragment must be a bool in [" + moduleName + "]")
		}
	}

	// 验证 payload_size 是 int
	if value, ok := icmpConfig["payload_size"]; ok {
		if _, ok := value.(int); !ok {
			return false, errors.New("ICMP_payload_size must be an integer in [" + moduleName + "]")
		}
	}

	// 验证 ttl 是 int，并且在 [0, 255] 范围内
	if value, ok := icmpConfig["ttl"]; ok {
		ttl, ok := value.(int)
		if !ok {
			return false, errors.New("ICMP_ttl must be an integer in [" + moduleName + "]")
		}
		if ttl < 0 || ttl > 255 {
			return false, errors.New("ICMP_ttl must be in the range [0, 255] in [" + moduleName + "]")
		}
	}

	return true, nil
}
