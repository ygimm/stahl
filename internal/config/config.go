package config

import "time"

type ServiceConfig struct {
}

type SchemaConfig struct {
	TableNames                 []string
	CreationTime               time.Time
	ChangelogTableNames        map[string]string
	ReplicationStatusTableName string
}

type UserSchemaConfig struct {
	TableNames []string
}

type ProducerConfig struct {
	Period      time.Duration
	StopOnError bool
	EnableDLQ   bool
}

type ConsumerConfig struct {
	MaxTasksBatch int
	MaxBatchWait  time.Duration
	StopOnError   bool
	EnableDLQ     bool
	ChannelName   string
}

type MetricsConfig struct {
}
