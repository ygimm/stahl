package pg

const (
	DropMethodEnumQuery = `
	drop type if exists stahl_method_enum
`
	CreateMethodEnumQuery = `
	create type stahl_method_enum as enum('INSERT', 'UPDATE', 'DELETE')
`
	CreateTableMetaDataTableQuery = `
	create table %s (
	    table_name text primary key, 
	    last_fetch_timestamp timestamp
	)
`
	CreateChangelogTableQuery = `
	create table %s (
	    id serial primary key,
		method stahl_method_enum not null ,
	    created_at timestamp default now(),
	    updated_at timestamp,
	    data jsonb not null default '{}'
	)
`
	CreateCreatedAtIdxForChangelogTableQuery = `
	create index if not exists 
		%s on %s (
			created_at
    );
`
)
