package config

type ServiceConfig struct {
}

type SchemaConfig struct {
	TableNames                 []string
	ChangelogTableNames        map[string]string
	ReplicationStatusTableName string
}

type ProducerConfig struct {
}

type ConsumerConfig struct {
}

type MetricsConfig struct {
}
