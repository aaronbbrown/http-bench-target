package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	negronilogrus "github.com/meatballhat/negroni-logrus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/urfave/negroni"
	negroniprometheus "github.com/zbindenren/negroni-prometheus"
)

type CPUProfileConfig struct {
	Iterations int
	Sleep      time.Duration
}

func NewCPUProfileConfigFromRequest(r *http.Request) (CPUProfileConfig, error) {
	var profile CPUProfileConfig
	iterstr := r.URL.Query().Get("iterations")
	if len(iterstr) < 1 {
		iterstr = "1000"
	}

	iterations, err := strconv.Atoi(iterstr)
	if err != nil {
		return profile, err
	}

	sleepstr := r.URL.Query().Get("sleep")
	if len(sleepstr) < 1 {
		sleepstr = "0s"
	}

	sleep, err := time.ParseDuration(sleepstr)
	if err != nil {
		return profile, err
	}

	profile.Iterations = iterations
	profile.Sleep = sleep

	return profile, nil
}

func envToMap(env []string) map[string]string {
	m := make(map[string]string)

	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		m[parts[0]] = parts[1]
	}

	return m
}

func main() {
	var wg sync.WaitGroup
	var workers uint
	var latencyFn string

	flag.UintVar(&workers, "simulated-workers", 0, "simulated number of http workers to artificially queue requests. 0 disables.")
	flag.StringVar(&latencyFn, "latency-filename", "",
		"filename containing latencies (one per line, in ms) to artificially delay requests via GET /latency")
	flag.Parse()
	fmt.Printf("GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))

	nprom := negroniprometheus.NewMiddleware("http-bench-target")
	mux := http.NewServeMux()

	mux.HandleFunc("/environment", func(w http.ResponseWriter, r *http.Request) {
		resp := struct {
			Environment    map[string]string `json:"environment,omiempty"`
			RequestHeaders http.Header       `json:"request_headers,omitempty"`
		}{
			Environment:    envToMap(os.Environ()),
			RequestHeaders: r.Header,
		}

		b, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprint(w, string(b))
	})

	mux.HandleFunc("/cpu", func(w http.ResponseWriter, r *http.Request) {
		profile, err := NewCPUProfileConfigFromRequest(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
		}

		for i := 1; i < profile.Iterations; i++ {
			if i%1000 == 0 {
				runtime.Gosched()
			}
		}

		time.Sleep(profile.Sleep)
		fmt.Fprintf(w, "iterations=%d\n", profile.Iterations)
	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "OK")
	})

	mux.Handle("/metrics", promhttp.Handler())

	if latencyFn != "" {
		lg, err := NewLatencyGeneratorFromFile(latencyFn)
		if err != nil {
			log.Fatal(err)
		}

		mux.Handle("/latency", lg)
	}

	formatter := &logrus.TextFormatter{
		DisableColors:   true,
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339Nano,
	}
	logrus.SetFormatter(formatter)
	logrus.SetOutput(os.Stdout)
	logger := negronilogrus.NewCustomMiddleware(logrus.InfoLevel, formatter, "web")
	n := negroni.New(negroni.NewRecovery(), logger, nprom)

	if workers > 0 {
		queue := NewLimitedQueueMiddleware(workers)
		n.Use(queue)
	}

	n.UseHandler(mux)

	srv := &http.Server{Addr: ":9090", Handler: n}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			// ListenAndServe() always returns an error
			log.Fatalf("Httpserver: ListenAndServe() error: %v", err)
		}
	}()

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGINT, syscall.SIGTERM)
	<-sigC

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal(err) // failure/timeout shutting down the server gracefully
	}
	wg.Wait()

	log.Println("main: done. exiting")
}
