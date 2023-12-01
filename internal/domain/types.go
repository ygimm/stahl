package domain

type GeneratorDriverType int

const (
	Postgres = 1
)

var DriverNameToType = map[string]GeneratorDriverType{
	"pg":         Postgres,
	"postgres":   Postgres,
	"postgresql": Postgres,
	"postgre":    Postgres,
}

type OutputDriverType int

const (
	Kafka = 1
)

var OutDriverNameToType = map[string]OutputDriverType{
	"kafka": Kafka,
}
