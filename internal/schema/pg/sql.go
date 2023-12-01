package pg

const (
	DropMethodEnumQuery = `
	drop type if exists stahl_method_enum
`
	CreateMethodEnumQuery = `
	create type stahl_method_enum as enum('INSERT', 'UPDATE', 'DELETE')
`
)

const (
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
	CreateDlqTableQuery = `
	create table %s (
	    id serial primary key,
	    origin_table text not null, 
	    payload jsonb,
	    origin_error text
	)
`
	DropTriggerQuery = `
	drop trigger if exists %s on %s
`
	DropProcedureQuery = `
	drop procedure if exists %s
`
	DropTableQuery = `
	drop table if exists %s
`
)

const (
	CreateCommonTriggerProcedureQuery = `
	create or replace function %s() returns trigger as $%s_changelog_audit$
    begin 
        if (tg_op = 'DELETE') then
            insert into %s (method, created_at, data) VALUES (TG_OP::stahl_method_enum, now(), row_to_json(OLD));
            return OLD;
        else
            insert into %s (method, created_at, data) VALUES (TG_OP::stahl_method_enum, now(), row_to_json(NEW));
            return OLD;
        end if;
    end;
    $%s_changelog_audit$ LANGUAGE plpgsql
`

	CreateTriggerForReplicaTableQuery = `
	create or replace trigger %s
    after insert or update or delete on %s
    for each row execute procedure %s();
`
)
