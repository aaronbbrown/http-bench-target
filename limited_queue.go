package main

import (
	"net/http"
	"strconv"

	"github.com/aaronbbrown/http-bench-target/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// LimitedQueueMiddlware holds requests until there's less than workers active
// requests in flight
type LimitedQueueMiddleware struct {
	// set workers to 0 to disable the queue
	workers uint
	acceptC chan bool
}

func NewLimitedQueueMiddleware(workers uint) *LimitedQueueMiddleware {
	return &LimitedQueueMiddleware{
		workers: workers,
		acceptC: make(chan bool, workers),
	}
}

func (m *LimitedQueueMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if m.workers > 0 {
		defer func() { <-m.acceptC }()
		timer := prometheus.NewTimer(metrics.QueueDurationSecs().WithLabelValues())
		m.acceptC <- true
		// measure how long we spent in the queue
		tiq := timer.ObserveDuration()
		logrus.WithFields(
			logrus.Fields{"milliseconds": tiq.Milliseconds()},
		).Info("time in queue")
		w.Header().Set("time-in-queue-ms", strconv.FormatInt(tiq.Milliseconds(), 10))
	}

	next(w, r)
}
