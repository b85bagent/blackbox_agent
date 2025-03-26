package module

import (
	"errors"
	"log"
	"time"
)

func validateHTTPModule(httpConfigCheck map[string]interface{}, moduleName string) (bool, error) {

	httpConfig, ok := httpConfigCheck["http"].(map[interface{}]interface{})
	if !ok {
		log.Println(httpConfigCheck)
		return false, errors.New("HTTP_prober_name error in [" + moduleName + "]")
	}

	if timeout, ok := httpConfig["timeout"]; ok {
		if _, err := time.ParseDuration(timeout.(string)); err != nil {
			return false, errors.New("HTTP_timeout must be a string in [" + moduleName + "]")
		}
	}

	if httpParams, ok := httpConfig["http"].(map[interface{}]interface{}); ok {
		if validHTTPVersions, ok := httpParams["valid_http_versions"].([]interface{}); ok {
			for _, version := range validHTTPVersions {
				if _, ok := version.(string); !ok {
					return false, errors.New("HTTP_valid_http_versions_data must be a string in [" + moduleName + "]")
				}
			}
		}

		if validStatusCodes, ok := httpParams["valid_status_codes"].([]interface{}); ok {
			for _, code := range validStatusCodes {
				if _, ok := code.(int); !ok {
					return false, errors.New("HTTP_valid_status_codes must be a int in [" + moduleName + "]")
				}
			}
		}

		if method, ok := httpParams["method"].(string); ok {
			validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
			if !contains(validMethods, method) {
				return false, errors.New("HTTP_method must be one of GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS in [" + moduleName + "]")
			}
		}

		if headers, ok := httpParams["headers"].(map[interface{}]interface{}); ok {
			for _, value := range headers {
				if _, ok := value.(string); !ok {
					return false, errors.New("HTTP_headers must be a map of strings in [" + moduleName + "]")
				}
			}
		}

		if _, ok := httpParams["follow_redirects"].(bool); !ok {
			return false, errors.New("HTTP_follow_redirects must be a boolean in [" + moduleName + "]")
		}

		if _, ok := httpParams["fail_if_ssl"].(bool); !ok {
			return false, errors.New("HTTP_fail_if_ssl must be a boolean in [" + moduleName + "]")
		}

		if _, ok := httpParams["fail_if_not_ssl"].(bool); !ok {
			return false, errors.New("HTTP_fail_if_not_ssl must be a boolean in [" + moduleName + "]")
		}

		if failIfBodyMatchesRegexp, ok := httpParams["fail_if_body_matches_regexp"].([]interface{}); ok {
			for _, regexp := range failIfBodyMatchesRegexp {
				if _, ok := regexp.(string); !ok {
					return false, errors.New("HTTP_each item in fail_if_body_matches_regexp must be a string in [" + moduleName + "]")
				}
			}
		}

		if failIfBodyNotMatchesRegexp, ok := httpParams["fail_if_body_not_matches_regexp"].([]interface{}); ok {
			for _, regexp := range failIfBodyNotMatchesRegexp {
				if _, ok := regexp.(string); !ok {
					return false, errors.New("HTTP_each item in fail_if_body_not_matches_regexp must be a string in [" + moduleName + "]")
				}
			}
		}

		if failIfHeaderMatches, ok := httpParams["fail_if_header_matches"].([]interface{}); ok {
			for _, matchSpec := range failIfHeaderMatches {
				if _, ok := matchSpec.(map[interface{}]interface{}); !ok {
					return false, errors.New("HTTP_each item in fail_if_header_matches must be a map in [" + moduleName + "]")
				}
			}
		}

		if failIfHeaderNotMatches, ok := httpParams["fail_if_header_not_matches"].([]interface{}); ok {
			for _, matchSpec := range failIfHeaderNotMatches {
				if _, ok := matchSpec.(map[interface{}]interface{}); !ok {
					return false, errors.New("HTTP_each item in fail_if_header_not_matches must be a map in [" + moduleName + "]")
				}
			}
		}

		if value, ok := httpParams["tls_config"]; ok {
			tlsConfig, ok := value.(map[interface{}]interface{})
			if !ok {
				return false, errors.New("HTTP_tls_config must be a map in [" + moduleName + "]")
			}
			if valid, err := validateTLSConfig(tlsConfig); !valid {
				return false, err
			}
		}

		if basicAuth, ok := httpParams["basic_auth"].(map[interface{}]interface{}); ok {
			if _, ok := basicAuth["username"].(string); !ok {
				return false, errors.New("HTTP_username in basic_auth must be a string in [" + moduleName + "]")
			}
			if _, ok := basicAuth["password"].(string); !ok {
				return false, errors.New("HTTP_password in basic_auth must be a string in [" + moduleName + "]")
			}
		}

		if authorization, ok := httpParams["authorization"].(map[interface{}]interface{}); ok {
			if _, ok := authorization["type"].(string); !ok {
				return false, errors.New("HTTP_type in authorization must be a string in [" + moduleName + "]")
			}
			if _, ok := authorization["credentials"].(string); !ok {
				return false, errors.New("HTTP_credentials in authorization must be a string in [" + moduleName + "]")
			}
			if _, ok := authorization["credentials_file"].(string); !ok {
				return false, errors.New("HTTP_credentials_file in authorization must be a string in [" + moduleName + "]")
			}
		}

		if proxyURL, ok := httpParams["proxy_url"].(string); ok && proxyURL == "" {
			return false, errors.New("HTTP_proxy_url must be a non-empty string in [" + moduleName + "]")
		}

		if noProxy, ok := httpParams["no_proxy"].(string); ok && noProxy == "" {
			return false, errors.New("HTTP_no_proxy must be a non-empty string in [" + moduleName + "]")
		}

		if _, ok := httpParams["proxy_from_environment"].(bool); !ok {
			return false, errors.New("HTTP_proxy_from_environment must be a boolean in [" + moduleName + "]")
		}

		if proxyConnectHeaders, ok := httpParams["proxy_connect_headers"].(map[interface{}]interface{}); ok {
			for _, value := range proxyConnectHeaders {
				if _, ok := value.(string); !ok {
					return false, errors.New("HTTP_each value in proxy_connect_headers must be a string in [" + moduleName + "]")
				}
			}
		}

		if _, ok := httpParams["skip_resolve_phase_with_proxy"].(bool); !ok {
			return false, errors.New("HTTP_skip_resolve_phase_with_proxy must be a boolean in [" + moduleName + "]")
		}

		if oauth2, ok := httpParams["oauth2"].(map[interface{}]interface{}); ok {

			if valid, err := validateOAuth2Config(oauth2); !valid {
				return false, err
			}

		}

		if _, ok := httpParams["enable_http2"].(bool); !ok {
			return false, errors.New("HTTP_enable_http2 must be a boolean in [" + moduleName + "]")
		}

		if preferredIPProtocol, ok := httpParams["preferred_ip_protocol"].(string); ok {
			validProtocols := []string{"ip4", "ip6"}
			if !contains(validProtocols, preferredIPProtocol) {
				return false, errors.New("HTTP_preferred_ip_protocol must be one of ip4, ip6 in [" + moduleName + "]")
			}
		}

		if _, ok := httpParams["ip_protocol_fallback"].(bool); !ok {
			return false, errors.New("HTTP_ip_protocol_fallback must be a boolean in [" + moduleName + "]")
		}

		if body, ok := httpParams["body"].(string); ok && body == "" {
			return false, errors.New("HTTP_body must be a non-empty string in [" + moduleName + "]")
		}

		if bodyFile, ok := httpParams["body_file"].(string); ok && bodyFile == "" {
			return false, errors.New("HTTP_body_file must be a non-empty string in [" + moduleName + "]")
		}

	}

	return true, nil
}

func contains(slice []string, element string) bool {
	for _, elem := range slice {
		if elem == element {
			return true
		}
	}
	return false
}

func validateTLSConfig(tlsConfig map[interface{}]interface{}) (bool, error) {
	// 驗證 insecure_skip_verify 是 boolean
	if value, ok := tlsConfig["insecure_skip_verify"]; ok {
		if _, ok := value.(bool); !ok {
			return false, errors.New("HTTP_insecure_skip_verify must be a boolean")
		}
	}

	// 驗證 ca_file 是 string
	if value, ok := tlsConfig["ca_file"]; ok {
		if _, ok := value.(string); !ok {
			return false, errors.New("HTTP_ca_file must be a string")
		}
	}

	// 驗證 cert_file 是 string
	if value, ok := tlsConfig["cert_file"]; ok {
		if _, ok := value.(string); !ok {
			return false, errors.New("HTTP_cert_file must be a string")
		}
	}

	// 驗證 key_file 是 string
	if value, ok := tlsConfig["key_file"]; ok {
		if _, ok := value.(string); !ok {
			return false, errors.New("HTTP_key_file must be a string")
		}
	}

	// 驗證 server_name 是 string
	if value, ok := tlsConfig["server_name"]; ok {
		if _, ok := value.(string); !ok {
			return false, errors.New("HTTP_server_name must be a string")
		}
	}

	// 驗證 min_version 的值是其中一個允許的值
	if value, ok := tlsConfig["min_version"]; ok {
		minVersion, ok := value.(string)
		if !ok {
			return false, errors.New("HTTP_min_version must be a string")
		}
		allowedValues := []string{"TLS10", "TLS11", "TLS12", "TLS13"}
		isValid := false
		for _, allowedValue := range allowedValues {
			if minVersion == allowedValue {
				isValid = true
				break
			}
		}
		if !isValid {
			return false, errors.New("HTTP_min_version is not a valid value")
		}
	}

	// 驗證 max_version 的值是其中一個允許的值
	if value, ok := tlsConfig["max_version"]; ok {
		maxVersion, ok := value.(string)
		if !ok {
			return false, errors.New("HTTP_max_version must be a string")
		}
		allowedValues := []string{"TLS10", "TLS11", "TLS12", "TLS13"}
		isValid := false
		for _, allowedValue := range allowedValues {
			if maxVersion == allowedValue {
				isValid = true
				break
			}
		}
		if !isValid {
			return false, errors.New("HTTP_max_version is not a valid value")
		}
	}

	return true, nil
}

func validateOAuth2Config(oauth2Config map[interface{}]interface{}) (bool, error) {
	// 驗證 client_id 是 string
	if value, ok := oauth2Config["client_id"]; ok {
		if _, ok := value.(string); !ok {
			return false, errors.New("HTTP_client_id must be a string")
		}
	} else {
		return false, errors.New("HTTP_client_id is required")
	}

	// 驗證 client_secret 和 client_secret_file 互斥
	clientSecret, hasClientSecret := oauth2Config["client_secret"]
	clientSecretFile, hasClientSecretFile := oauth2Config["client_secret_file"]
	if hasClientSecret && hasClientSecretFile {
		return false, errors.New("HTTP_client_secret and client_secret_file are mutually exclusive")
	}

	// 驗證 client_secret 是 string
	if hasClientSecret {
		if _, ok := clientSecret.(string); !ok {
			return false, errors.New("HTTP_client_secret must be a string")
		}
	}

	// 驗證 client_secret_file 是 string
	if hasClientSecretFile {
		if _, ok := clientSecretFile.(string); !ok {
			return false, errors.New("HTTP_client_secret_file must be a string")
		}
	}

	// 驗證 scopes 是 string 列表
	if value, ok := oauth2Config["scopes"]; ok {
		scopes, ok := value.([]interface{})
		if !ok {
			return false, errors.New("HTTP_scopes must be a list of strings")
		}
		for _, scope := range scopes {
			if _, ok := scope.(string); !ok {
				return false, errors.New("HTTP_each scope must be a string")
			}
		}
	}

	// 驗證 token_url 是 string
	if value, ok := oauth2Config["token_url"]; ok {
		if _, ok := value.(string); !ok {
			return false, errors.New("HTTP_token_url must be a string")
		}
	} else {
		return false, errors.New("HTTP_token_url is required")
	}

	// 驗證 endpoint_params 是 string 到 string 的映射
	if value, ok := oauth2Config["endpoint_params"]; ok {
		endpointParams, ok := value.(map[interface{}]interface{})
		if !ok {
			return false, errors.New("HTTP_endpoint_params must be a map of strings to strings")
		}
		for _, v := range endpointParams {
			if _, ok := v.(string); !ok {
				return false, errors.New("HTTP_each value in endpoint_params must be a string")
			}
		}
	}

	return true, nil
}
