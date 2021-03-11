package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
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
		DisableColors: true,
		FullTimestamp: true,
	}
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
		if err := srv.ListenAndServe(); err != nil {
			// cannot panic, because this probably is an intentional close
			log.Printf("Httpserver: ListenAndServe() error: %s", err)
		}
	}()

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGINT, syscall.SIGTERM)
	<-sigC

	// now close the server gracefully ("shutdown")
	// timeout could be given instead of nil as a https://golang.org/pkg/context/
	if err := srv.Shutdown(nil); err != nil {
		log.Fatal(err) // failure/timeout shutting down the server gracefully
	}
	wg.Wait()

	log.Println("main: done. exiting")
}
