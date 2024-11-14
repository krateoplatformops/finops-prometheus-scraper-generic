package apis

import (
	finopsDataTypes "github.com/krateoplatformops/finops-data-types/api/v1"
)

type ConfigFromFile struct {
	DatabaseConfigRef finopsDataTypes.ObjectRef `yaml:"databaseConfigRef"`
	Exporter          Exporter                  `yaml:"exporter"`
}

type Config struct {
	DatabaseConfig DatabaseConfig `yaml:"databaseConfig"`
	Exporter       Exporter       `yaml:"exporter"`
}

type DatabaseConfig struct {
	Username          string                            `yaml:"username" json:"username"`
	PasswordSecretRef finopsDataTypes.SecretKeySelector `yaml:"passwordSecretRef" json:"passwordSecretRef"`
}

type Exporter struct {
	Url                  string `yaml:"url"`
	PollingIntervalHours int    `yaml:"pollingIntervalHours"`
	TableName            string `yaml:"tableName"`
	MetricType           string `yaml:"metricType"`
}

type MetricRecord struct {
	Labels    map[string]string `json:"labels"`
	Value     float64           `json:"value"`
	Timestamp int64             `json:"timestamp"`
}

type UsernamePassword struct {
	Username string
	Password string
}
