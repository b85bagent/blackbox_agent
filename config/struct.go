package config

type PrometheusConfig struct {
	PrometheusUrl  string `yaml:"prometheusUrl"`
	Username       string `yaml:"username"`
	Password       string `yaml:"password"`
	PrometheusCert string `yaml:"prometheusCert"`
	EnableTLS      bool   `yaml:"enable_tls"`
}

type Config struct {
	Opensearch OpensearchSettings     `yaml:"opensearch"`
	RabbitMQ   RabbitMQSettings       `yaml:"rabbitMQ"`
	Prometheus PrometheusConfig       `yaml:"prometheus"`
	Const      map[string]interface{} `yaml:"const"`
}

type OpensearchSettings struct {
	Index      string                      `yaml:"index"`
	Enable     bool                        `yaml:"enable"`
	Opensearch map[string]OpensearchConfig `yaml:",inline"`
}
type OpensearchConfig struct {
	Host     []string `yaml:"host"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
}

type RabbitMQSettings struct {
	RabbitMQ map[string]RabbitMQConfig `yaml:",inline"`
}

type RabbitMQConfig struct {
	Host               []string `yaml:"host"`
	Username           string   `yaml:"username"`
	Password           string   `yaml:"password"`
	RabbitMQExchange   string   `yaml:"RabbitMQExchange"`
	RabbitMQRoutingKey string   `yaml:"RabbitMQRoutingKey"`
	RabbitMQQueue      []string `yaml:"RabbitMQQueue"`
}
