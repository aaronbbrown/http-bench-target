package main

import (
	"log"
	"net/http"
	"time"
)

// LimitedQueueMiddlware holds requests until there's less than workers active
// requests in flight
type LimitedQueueMiddleware struct {
	acceptC chan bool
}

func NewLimitedQueueMiddleware(workers int) *LimitedQueueMiddleware {
	return &LimitedQueueMiddleware{
		acceptC: make(chan bool, workers),
	}
}

func (m *LimitedQueueMiddleware) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	defer func() { <-m.acceptC }()
	t1 := time.Now()
	m.acceptC <- true
	// measure how long we spent in the queue
	tiq := time.Since(t1)
	log.Println("time in queue:", tiq)

	next(rw, r)
}
