package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CecileRobertMichon/reverse-proxy/internal/proxy"
)

func main() {
	var (
		address   string
		targetURL string
		timeout   time.Duration
	)

	flag.StringVar(&address, "address", "127.0.0.1:8080", "address for reverse proxy to listen on")
	flag.StringVar(&targetURL, "target", "https://www.boredapi.com/api/activity", "origin server to which the proxy should forward requests")
	flag.DurationVar(&timeout, "timeout", 10*time.Second, "timeout for requests to the origin server")

	flag.Parse()

	server := proxy.NewServer(targetURL, address, timeout)

	// Create a channel to receive OS signals
	done := make(chan os.Signal, 1)
	// Notify the 'done' channel for SIGINT or SIGTERM signals (Ctrl+C or `kill`)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	// Start a goroutine that will wait for a signal and then
	// gracefully shut down the server
	go func() {
		<-done
		fmt.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			fmt.Printf("Error shutting down server: %v\n", err)
		}
	}()

	log.Println("Starting reverse proxy server on address", address, "with target URL", targetURL)

	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
