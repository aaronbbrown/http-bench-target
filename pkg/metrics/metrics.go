package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func init() {
	queueDurationSecs = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "queue_duration",
			Help:       "Time in queue",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
		},
		[]string{},
	)
}

var (
	queueDurationSecs *prometheus.SummaryVec
)

func QueueDurationSecs() *prometheus.SummaryVec { return queueDurationSecs }
