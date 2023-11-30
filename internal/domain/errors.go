package domain

import "errors"

var (
	ErrorUnknownDriverName = errors.New("provided driver name not found")
	ErrorNoTablesSpecified = errors.New("no tables specified for replication")
)

var (
	ErrorUnknownChannel = errors.New("no channel specified with provided name")
)

var (
	ErrorTablesDoMatchWithSchema = errors.New("provided table names do not match with tables in actual schema")
)
