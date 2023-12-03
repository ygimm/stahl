package metrics

import (
	"time"

	prommetrics "github.com/deathowl/go-metrics-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rcrowley/go-metrics"
)

type IMetrics interface {
	SuccessPushEvent(table string)
	FailedPushEvent(table string)
	RegisterNew(r metrics.Registry)
}

type Metrics struct {
}

func (m Metrics) RegisterNew(r metrics.Registry) {
	client := prommetrics.NewPrometheusProvider(r, "sarama", "kafka", prometheus.DefaultRegisterer, 1*time.Second)
	go client.UpdatePrometheusMetrics()
}

func (m Metrics) SuccessPushEvent(table string) {
	successEvents.WithLabelValues(table).Inc()
}

func (m Metrics) FailedPushEvent(table string) {
	failedEvents.WithLabelValues(table).Inc()
}

func New() IMetrics {
	prometheus.MustRegister(successEvents, failedEvents)
	return &Metrics{}
}

var successEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "success_pushed_events_total",
}, []string{"table"})

var failedEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "failed_pushed_events_total",
}, []string{"table"})
