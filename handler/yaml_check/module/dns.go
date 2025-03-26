package module

import (
	"errors"
	"log"
)

func validateDNSConfig(dnsConfig map[string]interface{}, moduleName string) (bool, error) {

	dnsConfigCheck, ok := dnsConfig["dns"].(map[interface{}]interface{})
	if !ok {
		log.Println(dnsConfig)
		return false, errors.New("DNS_prober_name error in [" + moduleName + "]")
	}

	// 验证 query_name 是 string
	value, ok := dnsConfigCheck["query_name"]
	if !ok {
		return false, errors.New("DNS_query_name is required in [" + moduleName + "]")
	}
	if _, ok := value.(string); !ok {
		return false, errors.New("DNS_query_name must be a string in [" + moduleName + "]")
	}

	// 验证 preferred_ip_protocol 是 string
	if value, ok := dnsConfigCheck["preferred_ip_protocol"]; ok {
		if _, ok := value.(string); !ok {
			return false, errors.New("DNS_preferred_ip_protocol must be a string in [" + moduleName + "]")
		}
	}

	// 验证 ip_protocol_fallback 是 bool
	if value, ok := dnsConfigCheck["ip_protocol_fallback"]; ok {
		if _, ok := value.(bool); !ok {
			return false, errors.New("DNS_ip_protocol_fallback must be a bool in [" + moduleName + "]")
		}
	}

	// 验证 source_ip_address 是 string
	if value, ok := dnsConfigCheck["source_ip_address"]; ok {
		if _, ok := value.(string); !ok {
			return false, errors.New("DNS_source_ip_address must be a string in [" + moduleName + "]")
		}
	}

	// 验证 transport_protocol 是 string
	if value, ok := dnsConfigCheck["transport_protocol"]; ok {
		if _, ok := value.(string); !ok {
			return false, errors.New("DNS_transport_protocol must be a string in [" + moduleName + "]")
		}
	}

	// 验证 dns_over_tls 是 bool
	if value, ok := dnsConfigCheck["dns_over_tls"]; ok {
		if _, ok := value.(bool); !ok {
			return false, errors.New("DNS_dns_over_tls must be a bool in [" + moduleName + "]")
		}
	}

	if value, ok := dnsConfigCheck["tls_config"]; ok {
		tlsConfig, ok := value.(map[interface{}]interface{})
		if !ok {
			return false, errors.New("DNS_HTTP_tls_config must be a map in [" + moduleName + "]")
		}
		if valid, err := validateTLSConfig(tlsConfig); !valid {
			return false, err
		}
	}

	// 验证 query_type 是 string
	if value, ok := dnsConfigCheck["query_type"]; ok {
		if _, ok := value.(string); !ok {
			return false, errors.New("DNS_query_type must be a string in [" + moduleName + "]")
		}
	}

	// 验证 query_class 是 string
	if value, ok := dnsConfigCheck["query_class"]; ok {
		if _, ok := value.(string); !ok {
			return false, errors.New("DNS_query_class must be a string in [" + moduleName + "]")
		}
	}

	// 验证 recursion_desired 是 bool
	if value, ok := dnsConfigCheck["recursion_desired"]; ok {
		if _, ok := value.(bool); !ok {
			return false, errors.New("DNS_recursion_desired must be a bool in [" + moduleName + "]")
		}
	}

	// 验证 valid_rcodes 是 string 列表
	if value, ok := dnsConfigCheck["valid_rcodes"]; ok {
		validRcodes, ok := value.([]interface{})
		if !ok {
			return false, errors.New("DNS_valid_rcodes must be an array of strings in [" + moduleName + "]")
		}
		for _, rcode := range validRcodes {
			if _, ok := rcode.(string); !ok {
				return false, errors.New("DNS_each item in valid_rcodes must be a string in [" + moduleName + "]")
			}
		}
	}

	// 验证 validate_answer_rrs、validate_authority_rrs 和 validate_additional_rrs
	for _, key := range []string{"validate_answer_rrs", "validate_authority_rrs", "validate_additional_rrs"} {
		if value, ok := dnsConfigCheck[key]; ok {
			validateRRs, ok := value.(map[interface{}]interface{})
			if !ok {
				return false, errors.New(key + " must be a map in [" + moduleName + "]")
			}
			for _, subKey := range []string{"fail_if_matches_regexp", "fail_if_all_match_regexp", "fail_if_not_matches_regexp", "fail_if_none_matches_regexp"} {
				if subValue, ok := validateRRs[subKey]; ok {
					regexps, ok := subValue.([]interface{})
					if !ok {
						return false, errors.New(subKey + " in " + key + " must be an array of strings in [" + moduleName + "]")
					}
					for _, regexp := range regexps {
						if _, ok := regexp.(string); !ok {
							return false, errors.New("DNS_each item in " + subKey + " in " + key + " must be a string in [" + moduleName + "]")
						}
					}
				}
			}
		}
	}

	return true, nil
}
