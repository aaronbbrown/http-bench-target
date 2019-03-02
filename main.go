package main

import (
	crand "crypto/rand"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/urfave/negroni"
)

type CPUProfileConfig struct {
	Iterations int
	Sleep      time.Duration
}

func NewCPUProfileConfigFromRequest(r *http.Request) (CPUProfileConfig, error) {
	var profile CPUProfileConfig
	iterstr := r.URL.Query().Get("iterations")
	if len(iterstr) < 1 {
		iterstr = "100000"
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

func thread_cpu_usage() time.Duration {
	r := syscall.Rusage{}
	syscall.Getrusage(syscall.RUSAGE_SELF, &r)
	return time.Duration(r.Stime.Nano() + r.Utime.Nano())
}

func main() {
	fmt.Printf("GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))
	mux := http.NewServeMux()

	mux.HandleFunc("/cpu", func(w http.ResponseWriter, r *http.Request) {
		cpustart := thread_cpu_usage()

		profile, err := NewCPUProfileConfigFromRequest(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
		}

		for i := 1; i < profile.Iterations; i++ {
			_, err := crand.Int(crand.Reader, big.NewInt(27))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		time.Sleep(profile.Sleep)
		cpuend := thread_cpu_usage()
		fmt.Fprintf(w, "iterations=%d cputime=%d\n", profile.Iterations, cpuend.Nanoseconds()-cpustart.Nanoseconds())

	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "OK")
	})

	n := negroni.Classic()
	n.UseHandler(mux)

	srv := &http.Server{Addr: ":9090", Handler: n}

	go func() {
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

	log.Println("main: done. exiting")
}
