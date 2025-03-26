package target

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

const (
	blackboxFilename = "./blackbox_exporter/blackbox.yaml"
	targetFilename   = "./yaml/target.yaml"
)

type Configuration struct {
	ScrapeConfigs []ScrapeConfig `yaml:"scrape_configs"`
}

type ScrapeConfig struct {
	JobName        string         `yaml:"job_name"`
	ScrapeInterval string         `yaml:"scrape_interval"`
	MetricsPath    string         `yaml:"metrics_path"`
	Params         Params         `yaml:"params"`
	StaticConfigs  []StaticConfig `yaml:"static_configs"`
}

type Params struct {
	Module []interface{} `yaml:"module"`
}

type StaticConfig struct {
	Targets []string `yaml:"targets"`
	Labels  Labels   `yaml:"labels"`
	Tag     string   `yaml:"tag"`
}

type Labels struct {
	Check string `yaml:"check"`
}

func CheckTargetFormatAndReplace(data []byte) (bool, error) {
	if len(data) == 0 {
		return false, errors.New("input data is empty")
	}
	var config Configuration
	err := yaml.Unmarshal(data, &config)
	if err != nil {
		return false, fmt.Errorf("error unmarshalling the yaml: %w", err)
	}

	blackboxModules, err := loadBlackboxModules(blackboxFilename)
	if err != nil {
		return false, fmt.Errorf("error loading blackbox modules: %w", err)
	}

	if len(config.ScrapeConfigs) == 0 {
		return false, errors.New("missing params fields in the scrape_configs")
	}

	var validScrapeConfigs []ScrapeConfig
	var missingTargets []string

	for _, scrape := range config.ScrapeConfigs {
		if scrape.JobName == "" || scrape.ScrapeInterval == "" || scrape.MetricsPath == "" {
			return false, errors.New("missing mandatory fields in one of the scrape_configs")
		}
		if scrape.Params.Module == nil || len(scrape.Params.Module) == 0 {
			return false, errors.New(scrape.JobName + " missing 'module' in 'params' of one of the scrape_configs")
		}
		for _, module := range scrape.Params.Module {
			if _, ok := module.(string); !ok {
				return false, errors.New(scrape.JobName + " 'module' value in 'params' is not a string")
			}
			if !checkModuleExistence(module.(string), blackboxModules) {
				return false, fmt.Errorf("%s module :[%s] does not exist in blackbox.yaml", scrape.JobName, module)
			}
		}

		if len(scrape.StaticConfigs) == 0 {
			missingTargets = append(missingTargets, scrape.JobName)
			// 如果 StaticConfigs 为空，则跳过这个 scrape
			continue
		}

		var validStaticConfigs []StaticConfig
		for _, staticConfig := range scrape.StaticConfigs {
			if len(staticConfig.Targets) == 0 {
				// 如果 Targets 为空，则跳过这个 staticConfig
				missingTargets = append(missingTargets, scrape.JobName)
				continue
			}

			// ... (验证其他字段的代码)
			if staticConfig.Labels != (Labels{}) && len(staticConfig.Labels.Check) == 0 {
				return false, errors.New(scrape.JobName + " missing 'check' in 'labels' of one of the static_configs")
			}

			validStaticConfigs = append(validStaticConfigs, staticConfig)
		}

		if len(validStaticConfigs) == 0 {
			// 如果所有的 StaticConfigs 都不合法，则跳过这个 scrape
			continue
		}

		scrape.StaticConfigs = validStaticConfigs
		validScrapeConfigs = append(validScrapeConfigs, scrape)
	}

	if len(missingTargets) > 0 {
		if len(missingTargets) == len(config.ScrapeConfigs) {
			return false, fmt.Errorf("no matching static_configs found after validation")
		}
		config.ScrapeConfigs = validScrapeConfigs
		updatedData, err := yaml.Marshal(config)
		if err != nil {
			return false, fmt.Errorf("error marshalling the updated yaml: %w", err)
		}

		// Update the YAML file
		err = os.WriteFile(targetFilename, updatedData, 0666)
		if err != nil {
			return false, fmt.Errorf("error writing the updated yaml to file: %w", err)
		}

		if len(config.ScrapeConfigs) == 0 || len(validScrapeConfigs) == 0 {
			return false, fmt.Errorf("scrape_configs is empty")
		}

		return true, fmt.Errorf("missing 'targets' in static_configs of the following jobs: %v", missingTargets)
	}

	return true, nil
}
