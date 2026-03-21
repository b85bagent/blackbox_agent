package module

import (
	"errors"
	"log"
)

func validateTCPConfig(tcpConfigCheck map[string]interface{}, moduleName string) (bool, error) {

	tcpConfig, ok := tcpConfigCheck["tcp"].(map[interface{}]interface{})
	if !ok {
		log.Println(tcpConfigCheck)
		return false, errors.New("TCP_prober_name error in [" + moduleName + "]")
	}

	// 验证 preferred_ip_protocol 是 string
	if value, ok := tcpConfig["preferred_ip_protocol"]; ok {
		if _, ok := value.(string); !ok {
			return false, errors.New("TCP_preferred_ip_protocol must be a string in [" + moduleName + "]")
		}
	}

	// 验证 ip_protocol_fallback 是 bool
	if value, ok := tcpConfig["ip_protocol_fallback"]; ok {
		if _, ok := value.(bool); !ok {
			return false, errors.New("TCP_ip_protocol_fallback must be a bool in [" + moduleName + "]")
		}
	}

	// 验证 source_ip_address 是 string
	if value, ok := tcpConfig["source_ip_address"]; ok {
		if _, ok := value.(string); !ok {
			return false, errors.New("TCP_source_ip_address must be a string in [" + moduleName + "]")
		}
	}

	// 验证 query_response 是 slice
	if value, ok := tcpConfig["query_response"]; ok {
		responses, ok := value.([]interface{})
		if !ok {
			return false, errors.New("TCP_query_response must be a slice in [" + moduleName + "]")
		}
		for _, response := range responses {
			// 验证每个 response 是 map
			responseMap, ok := response.(map[interface{}]interface{})
			if !ok {
				log.Println("response: \n", response)
				return false, errors.New("TCP_each response in query_response must be a map in [" + moduleName + "]")
			}
			// 验证 expect 是 string
			if value, ok := responseMap["expect"]; ok {
				if _, ok := value.(string); !ok {
					return false, errors.New("TCP_expect in response must be a string in [" + moduleName + "]")
				}
			}
			if value, ok := responseMap["expect_bytes"]; ok {
				if _, ok := value.(string); !ok {
					return false, errors.New("TCP_expect_bytes in response must be a string in [" + moduleName + "]")
				}
			}
			// 验证 send 是 string
			if value, ok := responseMap["send"]; ok {
				if _, ok := value.(string); !ok {
					return false, errors.New("TCP_send in response must be a string in [" + moduleName + "]")
				}
			}
			// 验证 starttls 是 bool
			if value, ok := responseMap["starttls"]; ok {
				if _, ok := value.(bool); !ok {
					return false, errors.New("TCP_starttls in response must be a bool in [" + moduleName + "]")
				}
			}
		}
	}

	// 验证 tls 是 bool
	if value, ok := tcpConfig["tls"]; ok {
		if _, ok := value.(bool); !ok {
			return false, errors.New("TCP_tls must be a bool in [" + moduleName + "]")
		}
	}

	if value, ok := tcpConfig["tls_config"]; ok {
		tlsConfig, ok := value.(map[interface{}]interface{})
		if !ok {
			return false, errors.New("TCP_HTTP_tls_config must be a map in [" + moduleName + "]")
		}
		if valid, err := validateTLSConfig(tlsConfig); !valid {
			return false, err
		}
	}

	return true, nil
}
