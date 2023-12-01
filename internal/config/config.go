package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

func ParseConfig() (Summary, error) {
	var cfg Summary

	//data, err := os.ReadFile("./stahl/config/config.example.yaml")
	data, err := os.ReadFile("/Users/elarkin/Documents/GitHub/stahl/.stahl/config/config.example.yml")
	if err != nil {
		return cfg, fmt.Errorf("os.ReadFile: %w", err)
	}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("yaml.Unmarshal: %w", err)
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
	Schema   SchemaConfig   `yaml:"-"`
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
