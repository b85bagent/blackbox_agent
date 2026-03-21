package module

import (
	"errors"
	"log"
)

func validateGRPCConfig(grpcConfigCheck map[string]interface{}, moduleName string) (bool, error) {

	grpcConfig, ok := grpcConfigCheck["grpc"].(map[interface{}]interface{})
	if !ok {
		log.Println(grpcConfigCheck)
		return false, errors.New("GRPC_prober_name error in [" + moduleName + "]")
	}

	// 验证 service 是 string
	if value, ok := grpcConfig["service"]; ok {
		if _, ok := value.(string); !ok {
			return false, errors.New("GRPC_service must be a string in [" + moduleName + "]")
		}
	}

	// 验证 preferred_ip_protocol 是 string
	if value, ok := grpcConfig["preferred_ip_protocol"]; ok {
		if _, ok := value.(string); !ok {
			return false, errors.New("GRPC_preferred_ip_protocol must be a string in [" + moduleName + "]")
		}
	}

	// 验证 ip_protocol_fallback 是 bool
	if value, ok := grpcConfig["ip_protocol_fallback"]; ok {
		if _, ok := value.(bool); !ok {
			return false, errors.New("GRPC_ip_protocol_fallback must be a bool in [" + moduleName + "]")
		}
	}

	// 验证 tls 是 bool
	if value, ok := grpcConfig["tls"]; ok {
		if _, ok := value.(bool); !ok {
			return false, errors.New("GRPC_tls must be a bool in [" + moduleName + "]")
		}
	}

	if value, ok := grpcConfig["metadata"]; ok {
		metadataConfig, ok := value.(map[interface{}]interface{})
		if !ok {
			return false, errors.New("GRPC_metadata must be a map in [" + moduleName + "]")
		}
		for _, item := range metadataConfig {
			values, ok := item.([]interface{})
			if !ok {
				return false, errors.New("GRPC_metadata values must be string lists in [" + moduleName + "]")
			}
			for _, metadataValue := range values {
				if _, ok := metadataValue.(string); !ok {
					return false, errors.New("GRPC_metadata values must be string lists in [" + moduleName + "]")
				}
			}
		}
	}

	if value, ok := grpcConfig["tls_config"]; ok {
		tlsConfig, ok := value.(map[interface{}]interface{})
		if !ok {
			return false, errors.New("GRPC_HTTP_tls_config must be a map in [" + moduleName + "]")
		}
		if valid, err := validateTLSConfig(tlsConfig); !valid {
			return false, err
		}
	}

	return true, nil
}
