package module

import (
	"errors"
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

const (
	blackboxFilename = "./blackbox_exporter/blackbox.yaml"
)

type Module struct {
	Prober  string                 `yaml:"prober"`
	Timeout string                 `yaml:"timeout,omitempty"`
	Extra   map[string]interface{} `yaml:",inline"`
}

type Modules struct {
	ModuleList map[string]Module `yaml:"modules"`
}

func CheckModuleFormat(data []byte) (bool, error) {
	if len(data) == 0 {
		return false, errors.New("input data is empty")
	}
	var modules Modules
	err := yaml.Unmarshal(data, &modules)
	if err != nil {
		log.Println("Error unmarshalling the yaml:", err)
		return false, err
	}

	allowedProbers := []string{"http", "tcp", "grpc", "icmp", "dns"}

	// 建立一個新的模組列表，用於儲存非空模組
	validModules := make(map[string]Module)
	var missingTargets []string

	for moduleName, module := range modules.ModuleList {

		// 检查模块的有效性
		isValidProber := false
		for _, prober := range allowedProbers {
			if module.Prober == prober {
				isValidProber = true
				break
			}
		}

		if !isValidProber {
			log.Printf("Invalid prober '%s' in module '%s'\n", module.Prober, moduleName)
			return false, errors.New("Invalid prober " + module.Prober + " in module " + moduleName + "\n")
		}

		// 检查其他键是否有效
		for key := range module.Extra {
			if key != module.Prober && key != "timeout" {
				log.Printf("Invalid parameter '%s' in module '%s'. Expected only 'timeout' or parameters starting with '%s:'\n", key, moduleName, module.Prober)
				return false, errors.New("Invalid parameter " + key + " in module " + moduleName + ". Expected only 'timeout' or parameters starting with " + module.Prober + "\n")
			}
		}

		// 检查是否包含 prober 参数
		if module.Prober != "" {
			// 如果模块的 prober 对应的值为空，则移除该键值对
			if extraValue, ok := module.Extra[module.Prober]; ok {
				if isValueEmpty(extraValue) {
					delete(module.Extra, module.Prober)
					missingTargets = append(missingTargets, moduleName)
				}
			}

			// 检查模块是否为空
			if isEmptyModule(module) {
				validModules[moduleName] = module
				continue
			}
		}

		// 验证模块的其他规则
		validModule, errValid := validateModule(module, moduleName)
		if !validModule {
			log.Println("errValid: ", errValid)
			return false, errValid
		}

		// 将有效模块添加到新的模块列表中
		validModules[moduleName] = module
	}

	// 將非空模組列表重新設定為模組列表
	modules.ModuleList = validModules

	if len(missingTargets) > 0 || len(validModules) != len(modules.ModuleList) {

		// 將模組重新編碼為YAML
		yamlData, err := yaml.Marshal(&modules)
		if err != nil {
			log.Println("Error marshalling the modules to YAML:", err)
			return false, err
		}

		// 將更新後的YAML資料寫回原始文件
		err = os.WriteFile(blackboxFilename, yamlData, 0644)
		if err != nil {
			log.Println("Error writing updated YAML data to file:", err)
			return false, err
		}

		return true, fmt.Errorf("missing 'prober.value.value' in prober.value of the following jobs: %v", missingTargets)

	}

	return true, nil
}

func validateModule(module Module, moduleName string) (bool, error) {

	if value, ok := module.Extra[module.Prober]; ok && value != nil {
		switch value := value.(type) {
		case map[interface{}]interface{}:
			if value == nil || (len(value) == 0) {
				return false, errors.New("Empty map for key '" + module.Prober + "' in [ " + moduleName + " ]")
			}
		}
	}

	switch module.Prober {
	case "http":
		return validateHTTPModule(module.Extra, moduleName)
	case "dns":
		return validateDNSConfig(module.Extra, moduleName)
	case "icmp":
		return validateICMPConfig(module.Extra, moduleName)
	case "tcp":
		return validateTCPConfig(module.Extra, moduleName)
	case "grpc":
		return validateGRPCConfig(module.Extra, moduleName)
	default:
		return false, errors.New("Invalid Prober value " + module.Prober)
	}

}

// 检查模块是否为空
func isEmptyModule(module Module) bool {
	return len(module.Extra) == 0
}

// 檢查模組是否包含空的必需子模組
func hasEmptyRequiredExtraModule(module Module) bool {
	if value, ok := module.Extra[module.Prober]; ok && value != nil {
		switch value := value.(type) {
		case map[interface{}]interface{}:
			return len(value) == 0
		}
	}
	return false
}

// 检查值是否为空
func isValueEmpty(value interface{}) bool {
	switch value := value.(type) {
	case map[interface{}]interface{}:
		return len(value) == 0
	default:
		return value == nil
	}
}
