package config

import (
	"bufio"
	"errors"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

const filePath = "./yaml/"

func ConfigInit(configFile string) (*Config, error) {

	match, err := regexp.MatchString("^config.*\\.*", configFile)
	if err != nil {
		e := errors.New("config regexp error : " + err.Error())
		return nil, e
	}

	if !match {
		e := errors.New("config 檔案名稱不符合要求，請用-h 確認指令以及符合的yaml格式")
		return nil, e
	}

	configFile = filePath + configFile

	// 先對配置文件進行環境變量替換
	processedContent, err := processYAMLFile(configFile)
	if err != nil {
		return nil, err
	}

	// 定义一个Config类型的变量来存储解析后的配置信息
	var config Config

	// 解析配置文件
	if err = yaml.Unmarshal([]byte(processedContent), &config); err != nil {
		return nil, err
	}

	return &config, nil

}

func processYAMLFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var processedLines []string
	scanner := bufio.NewScanner(file)
	envVarRegex := regexp.MustCompile(`\$\{([^}]+)\}`)
	defaultValueRegex := regexp.MustCompile(`#\s*(.+)`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := envVarRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			envVar := matches[1]
			envValue, exists := os.LookupEnv(envVar)
			if !exists {
				// Check for default value in comment
				defaultMatches := defaultValueRegex.FindStringSubmatch(line)
				if len(defaultMatches) > 1 {
					envValue = defaultMatches[1] // Use the default value
				}
			}
			line = envVarRegex.ReplaceAllString(line, envValue)
		}
		processedLines = append(processedLines, line)
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.Join(processedLines, "\n"), nil
}
