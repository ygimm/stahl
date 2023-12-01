package storepg

var (
	PushDlqQuery = `
	insert into $1
	    (origin_table, payload, origin_error)
	values 
		($2, $3, $4)
	`
)

var (
	GetIdsQuery = `
	select repl.id from
	      %s repl join %s stat on stat.table_name=$1
	where repl.created_at >= stat.last_fetch_timestamp or stat.last_fetch_timestamp is NULL
`

	UpdateLastFetchTimestampQuery = `
	update %s
	set last_fetch_timestamp = now()
	where table_name = $1
`
)

var (
	GetEventsByIdsQuery = `
	
`
)
