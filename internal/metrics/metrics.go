package metrics

type IMetrics interface {
	SuccessPushTask() error
	SuccessFetchTask() error
	FailedPushTask() error
	FiledFetchTask() error
}
