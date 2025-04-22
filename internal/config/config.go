package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

func ParseConfig() (Summary, error) {
	var cfg Summary
	var err error
	var data []byte

	configPaths := []string{
		"/app/config/config.yml",
		"/app/config/config.example.yml",
		"./.stahl/config/config.yml",
		"./.stahl/config/config.example.yml",
	}

	// Попытка загрузить конфигурацию из всех возможных путей
	for _, path := range configPaths {
		log.Printf("Trying to load config from: %s", path)
		data, err = os.ReadFile(path)
		if err == nil {
			log.Printf("Successfully loaded config from: %s", path)
			break
		}
		log.Printf("Failed to load config from %s: %v", path, err)
	}

	if err != nil {
		return cfg, fmt.Errorf("failed to load config from any path: %w", err)
	}

	// Проверка переменных окружения на наличие переопределений для брокера Kafka
	if kafkaBrokers := os.Getenv("KAFKA_BROKERS"); kafkaBrokers != "" {
		log.Printf("Found KAFKA_BROKERS environment variable: %s", kafkaBrokers)
	}

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("yaml.Unmarshal: %w", err)
	}

	// Если есть переменная окружения KAFKA_BROKERS, она имеет приоритет над конфигурационным файлом
	if kafkaBrokers := os.Getenv("KAFKA_BROKERS"); kafkaBrokers != "" {
		cfg.Drivers.Output.Brokers = []string{kafkaBrokers}
		log.Printf("Using Kafka brokers from environment: %v", cfg.Drivers.Output.Brokers)
	} else {
		log.Printf("Using Kafka brokers from config file: %v", cfg.Drivers.Output.Brokers)
	}

	return cfg, nil
}

type Summary struct {
	UserSchema UserSchemaConfig `yaml:"schema"`
	Producer   ProducerConfig   `yaml:"producer"`
	Drivers    struct {
		Db     DatabaseConfig `yaml:"db"`
		Output OutputConfig   `yaml:"output"`
	} `yaml:"drivers"`
	Consumer ConsumerConfig `yaml:"consumer"`
	Service  ServiceConfig  `yaml:"service"`
	Schema   SchemaConfig   `yaml:"-"`
}

type ServiceConfig struct {
	MetricsPort uint16 `yaml:"metrics_port"`
}

type SchemaConfig struct {
	TableNames                 []string
	CreationTime               time.Time
	ChangelogTableNames        map[string]string
	ReplicationStatusTableName string
}

type UserSchemaConfig struct {
	TableNames []string `yaml:"tables"`
}

type ProducerConfig struct {
	Period      time.Duration `yaml:"period"`
	StopOnError bool          `yaml:"stop_on_error"`
	EnableDLQ   bool          `yaml:"enable_dlq"`
}

type ConsumerConfig struct {
	MaxTasksBatch int           `yaml:"max_tasks_batch"`
	MaxBatchWait  time.Duration `yaml:"max_batch_wait"`
	StopOnError   bool          `yaml:"stop_on_error"`
	EnableDLQ     bool          `yaml:"enable_dlq"`
	ChannelName   string
}

type DatabaseConfig struct {
	DriverName  string        `yaml:"driver_name"`
	Host        string        `yaml:"host"`
	Port        uint16        `yaml:"port"`
	User        string        `yaml:"user"`
	Password    string        `yaml:"password,omitempty"`
	Database    string        `yaml:"database"`
	PingTimeout time.Duration `yaml:"ping_timeout"`
	PingPeriod  time.Duration `yaml:"ping_period"`
}

type OutputConfig struct {
	DriverName   string            `yaml:"driver_name"`
	TableChannel map[string]string `yaml:"table_channel"`
	Brokers      []string          `yaml:"brokers"`
}
