package main

import (
	crand "crypto/rand"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/urfave/negroni"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/cpu", func(w http.ResponseWriter, r *http.Request) {
		iterations := 100000
		for i := 1; i < iterations; i++ {
			_, err := crand.Int(crand.Reader, big.NewInt(27))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		io.WriteString(w, "Generated "+strconv.Itoa(iterations)+" random strings")
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
