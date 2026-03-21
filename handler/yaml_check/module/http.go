package module

import (
	"errors"
	"log"
	"strings"
	"time"
)

func validateHTTPModule(httpConfigCheck map[string]interface{}, moduleName string) (bool, error) {
	httpConfig, ok := httpConfigCheck["http"].(map[interface{}]interface{})
	if !ok {
		log.Println(httpConfigCheck)
		return false, errors.New("HTTP_prober_name error in [" + moduleName + "]")
	}

	if timeout, ok := httpConfigCheck["timeout"]; ok {
		timeoutString, ok := timeout.(string)
		if !ok {
			return false, errors.New("HTTP_timeout must be a string in [" + moduleName + "]")
		}
		if _, err := time.ParseDuration(timeoutString); err != nil {
			return false, errors.New("HTTP_timeout must be a string in [" + moduleName + "]")
		}
	}

	if validHTTPVersions, ok := httpConfig["valid_http_versions"]; ok {
		if err := validateStringSlice(validHTTPVersions, "HTTP_valid_http_versions_data must be a string in ["+moduleName+"]"); err != nil {
			return false, err
		}
	}

	if validStatusCodes, ok := httpConfig["valid_status_codes"]; ok {
		if err := validateIntSlice(validStatusCodes, "HTTP_valid_status_codes must be a int in ["+moduleName+"]"); err != nil {
			return false, err
		}
	}

	if method, ok := httpConfig["method"]; ok {
		methodString, ok := method.(string)
		if !ok {
			return false, errors.New("HTTP_method must be one of GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS in [" + moduleName + "]")
		}
		validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
		if !contains(validMethods, strings.ToUpper(methodString)) {
			return false, errors.New("HTTP_method must be one of GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS in [" + moduleName + "]")
		}
	}

	if headers, ok := httpConfig["headers"]; ok {
		headersMap, ok := headers.(map[interface{}]interface{})
		if !ok {
			return false, errors.New("HTTP_headers must be a map of strings in [" + moduleName + "]")
		}
		for _, value := range headersMap {
			if _, ok := value.(string); !ok {
				return false, errors.New("HTTP_headers must be a map of strings in [" + moduleName + "]")
			}
		}
	}

	if err := validateOptionalBool(httpConfig, "follow_redirects", "HTTP_follow_redirects must be a boolean in ["+moduleName+"]"); err != nil {
		return false, err
	}

	if err := validateOptionalBool(httpConfig, "fail_if_ssl", "HTTP_fail_if_ssl must be a boolean in ["+moduleName+"]"); err != nil {
		return false, err
	}

	if err := validateOptionalBool(httpConfig, "fail_if_not_ssl", "HTTP_fail_if_not_ssl must be a boolean in ["+moduleName+"]"); err != nil {
		return false, err
	}

	if failIfBodyMatchesRegexp, ok := httpConfig["fail_if_body_matches_regexp"]; ok {
		if err := validateStringSlice(failIfBodyMatchesRegexp, "HTTP_each item in fail_if_body_matches_regexp must be a string in ["+moduleName+"]"); err != nil {
			return false, err
		}
	}

	if failIfBodyNotMatchesRegexp, ok := httpConfig["fail_if_body_not_matches_regexp"]; ok {
		if err := validateStringSlice(failIfBodyNotMatchesRegexp, "HTTP_each item in fail_if_body_not_matches_regexp must be a string in ["+moduleName+"]"); err != nil {
			return false, err
		}
	}

	if failIfHeaderMatches, ok := httpConfig["fail_if_header_matches"]; ok {
		if err := validateHeaderMatches(failIfHeaderMatches, moduleName, "HTTP_each item in fail_if_header_matches must be a map in ["); err != nil {
			return false, err
		}
	}

	if failIfHeaderNotMatches, ok := httpConfig["fail_if_header_not_matches"]; ok {
		if err := validateHeaderMatches(failIfHeaderNotMatches, moduleName, "HTTP_each item in fail_if_header_not_matches must be a map in ["); err != nil {
			return false, err
		}
	}

	if value, ok := httpConfig["tls_config"]; ok {
		tlsConfig, ok := value.(map[interface{}]interface{})
		if !ok {
			return false, errors.New("HTTP_tls_config must be a map in [" + moduleName + "]")
		}
		if valid, err := validateTLSConfig(tlsConfig); !valid {
			return false, err
		}
	}

	if basicAuth, ok := httpConfig["basic_auth"]; ok {
		basicAuthConfig, ok := basicAuth.(map[interface{}]interface{})
		if !ok {
			return false, errors.New("HTTP_basic_auth must be a map in [" + moduleName + "]")
		}
		if _, ok := basicAuthConfig["username"].(string); !ok {
			return false, errors.New("HTTP_username in basic_auth must be a string in [" + moduleName + "]")
		}
		if _, ok := basicAuthConfig["password"].(string); !ok {
			return false, errors.New("HTTP_password in basic_auth must be a string in [" + moduleName + "]")
		}
	}

	if authorization, ok := httpConfig["authorization"]; ok {
		authorizationConfig, ok := authorization.(map[interface{}]interface{})
		if !ok {
			return false, errors.New("HTTP_authorization must be a map in [" + moduleName + "]")
		}
		if value, exists := authorizationConfig["type"]; exists {
			if _, ok := value.(string); !ok {
				return false, errors.New("HTTP_type in authorization must be a string in [" + moduleName + "]")
			}
		}
		if value, exists := authorizationConfig["credentials"]; exists {
			if _, ok := value.(string); !ok {
				return false, errors.New("HTTP_credentials in authorization must be a string in [" + moduleName + "]")
			}
		}
		if value, exists := authorizationConfig["credentials_file"]; exists {
			if _, ok := value.(string); !ok {
				return false, errors.New("HTTP_credentials_file in authorization must be a string in [" + moduleName + "]")
			}
		}
	}

	if proxyURL, ok := httpConfig["proxy_url"]; ok {
		proxyURLString, ok := proxyURL.(string)
		if !ok || proxyURLString == "" {
			return false, errors.New("HTTP_proxy_url must be a non-empty string in [" + moduleName + "]")
		}
	}

	if noProxy, ok := httpConfig["no_proxy"]; ok {
		noProxyString, ok := noProxy.(string)
		if !ok || noProxyString == "" {
			return false, errors.New("HTTP_no_proxy must be a non-empty string in [" + moduleName + "]")
		}
	}

	if err := validateOptionalBool(httpConfig, "proxy_from_environment", "HTTP_proxy_from_environment must be a boolean in ["+moduleName+"]"); err != nil {
		return false, err
	}

	if proxyConnectHeaders, ok := httpConfig["proxy_connect_headers"]; ok {
		headersMap, ok := proxyConnectHeaders.(map[interface{}]interface{})
		if !ok {
			return false, errors.New("HTTP_proxy_connect_headers must be a map of strings in [" + moduleName + "]")
		}
		for _, value := range headersMap {
			if _, ok := value.(string); !ok {
				return false, errors.New("HTTP_each value in proxy_connect_headers must be a string in [" + moduleName + "]")
			}
		}
	}

	if err := validateOptionalBool(httpConfig, "skip_resolve_phase_with_proxy", "HTTP_skip_resolve_phase_with_proxy must be a boolean in ["+moduleName+"]"); err != nil {
		return false, err
	}

	if oauth2, ok := httpConfig["oauth2"]; ok {
		oauth2Config, ok := oauth2.(map[interface{}]interface{})
		if !ok {
			return false, errors.New("HTTP_oauth2 must be a map in [" + moduleName + "]")
		}
		if valid, err := validateOAuth2Config(oauth2Config); !valid {
			return false, err
		}
	}

	if err := validateOptionalBool(httpConfig, "enable_http2", "HTTP_enable_http2 must be a boolean in ["+moduleName+"]"); err != nil {
		return false, err
	}

	if err := validateOptionalBool(httpConfig, "enable_http3", "HTTP_enable_http3 must be a boolean in ["+moduleName+"]"); err != nil {
		return false, err
	}

	if preferredIPProtocol, ok := httpConfig["preferred_ip_protocol"]; ok {
		preferredIPProtocolString, ok := preferredIPProtocol.(string)
		if !ok {
			return false, errors.New("HTTP_preferred_ip_protocol must be one of ip4, ip6 in [" + moduleName + "]")
		}
		validProtocols := []string{"ip4", "ip6"}
		if !contains(validProtocols, preferredIPProtocolString) {
			return false, errors.New("HTTP_preferred_ip_protocol must be one of ip4, ip6 in [" + moduleName + "]")
		}
	}

	if err := validateOptionalBool(httpConfig, "ip_protocol_fallback", "HTTP_ip_protocol_fallback must be a boolean in ["+moduleName+"]"); err != nil {
		return false, err
	}

	if body, ok := httpConfig["body"]; ok {
		bodyString, ok := body.(string)
		if !ok || bodyString == "" {
			return false, errors.New("HTTP_body must be a non-empty string in [" + moduleName + "]")
		}
	}

	if bodyFile, ok := httpConfig["body_file"]; ok {
		bodyFileString, ok := bodyFile.(string)
		if !ok || bodyFileString == "" {
			return false, errors.New("HTTP_body_file must be a non-empty string in [" + moduleName + "]")
		}
	}

	if compression, ok := httpConfig["compression"]; ok {
		if _, ok := compression.(string); !ok {
			return false, errors.New("HTTP_compression must be a string in [" + moduleName + "]")
		}
	}

	if bodySizeLimit, ok := httpConfig["body_size_limit"]; ok {
		switch bodySizeLimit.(type) {
		case int, int64, uint64, float64, string:
		default:
			return false, errors.New("HTTP_body_size_limit must be a string or number in [" + moduleName + "]")
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

func validateOptionalBool(config map[interface{}]interface{}, key, errMessage string) error {
	if value, ok := config[key]; ok {
		if _, ok := value.(bool); !ok {
			return errors.New(errMessage)
		}
	}
	return nil
}

func validateStringSlice(value interface{}, errMessage string) error {
	values, ok := value.([]interface{})
	if !ok {
		return errors.New(errMessage)
	}
	for _, item := range values {
		if _, ok := item.(string); !ok {
			return errors.New(errMessage)
		}
	}
	return nil
}

func validateIntSlice(value interface{}, errMessage string) error {
	values, ok := value.([]interface{})
	if !ok {
		return errors.New(errMessage)
	}
	for _, item := range values {
		if _, ok := item.(int); !ok {
			return errors.New(errMessage)
		}
	}
	return nil
}

func validateHeaderMatches(value interface{}, moduleName, prefix string) error {
	matchSpecs, ok := value.([]interface{})
	if !ok {
		return errors.New(prefix + moduleName + "]")
	}
	for _, matchSpec := range matchSpecs {
		matchMap, ok := matchSpec.(map[interface{}]interface{})
		if !ok {
			return errors.New(prefix + moduleName + "]")
		}
		if header, exists := matchMap["header"]; exists {
			if _, ok := header.(string); !ok {
				return errors.New("HTTP_header in header match must be a string in [" + moduleName + "]")
			}
		}
		if regexp, exists := matchMap["regexp"]; exists {
			if _, ok := regexp.(string); !ok {
				return errors.New("HTTP_regexp in header match must be a string in [" + moduleName + "]")
			}
		}
		if allowMissing, exists := matchMap["allow_missing"]; exists {
			if _, ok := allowMissing.(bool); !ok {
				return errors.New("HTTP_allow_missing in header match must be a boolean in [" + moduleName + "]")
			}
		}
	}
	return nil
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
