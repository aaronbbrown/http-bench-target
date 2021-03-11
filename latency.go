package main

import (
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type LatencyGenerator struct {
	durations []time.Duration
	mutex     sync.Mutex
	index     int
}

func NewLatencyGeneratorFromFile(filename string) (*LatencyGenerator, error) {
	var durations []time.Duration
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// read in each whitespace delimited value and convert it to a millsecond
	// duration
	for _, str := range strings.Fields(string(data)) {
		ms, err := strconv.Atoi(str)
		if err != nil {
			return nil, err
		}

		durations = append(durations, time.Duration(ms)*time.Millisecond)
	}

	return &LatencyGenerator{
		durations: durations,
	}, nil
}

func (lg *LatencyGenerator) nextDuration() time.Duration {
	lg.mutex.Lock()
	defer lg.mutex.Unlock()

	// wrap around
	if lg.index >= len(lg.durations) {
		lg.index = 0
	}
	duration := lg.durations[lg.index]
	lg.index++
	return duration
}

func (lg *LatencyGenerator) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	duration := lg.nextDuration()
	time.Sleep(duration)
	w.Header().Set("selected-latency", duration.String())

	w.WriteHeader(http.StatusOK)
}
