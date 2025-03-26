package target

import (
	"os"

	"blackbox_agent/handler/yaml_check/module"

	"gopkg.in/yaml.v2"
)

func loadBlackboxModules(filename string) ([]string, error) {
	// 读取yaml文件
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config module.Modules
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	modules := make([]string, 0, len(config.ModuleList))
	for moduleName := range config.ModuleList {
		modules = append(modules, moduleName)
	}

	return modules, nil
}

func checkModuleExistence(moduleName string, blackboxModules []string) bool {
	for _, m := range blackboxModules {
		if moduleName == m {
			return true
		}
	}
	return false
}
