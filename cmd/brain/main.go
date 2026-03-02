// Brain server - runs Koji's emotional state machine and exposes it via HTTP API.
// This is designed to run on a server, with the Pi and ESP32 as clients.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/alex/koji/internal/api"
	"github.com/alex/koji/internal/brain"
)

func main() {
	// Flags
	apiAddr := flag.String("addr", ":8080", "API server address")
	flag.Parse()

	log.Println("=== Koji Brain Server ===")

	// Create the brain
	cfg := brain.DefaultConfig()
	b := brain.New(cfg)

	// Create and wire up the API server
	server := api.NewServer(*apiAddr, b, b)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the brain's main loop in background
	go func() {
		if err := b.Run(ctx); err != nil && err != context.Canceled {
			log.Printf("Brain error: %v", err)
		}
	}()

	// Start API server in background
	go func() {
		if err := server.Start(ctx); err != nil {
			log.Printf("API server error: %v", err)
			cancel()
		}
	}()

	log.Printf("Koji brain ready. Listening on %s", *apiAddr)
	log.Println()
	log.Println("Endpoints:")
	log.Println("  GET  /api/state  - get current emotional state")
	log.Println("  POST /api/event  - send sensor event")
	log.Println("  GET  /health     - health check")
	log.Println()

	// Wait for signal
	sig := <-sigChan
	log.Printf("Received signal %v, shutting down...", sig)
	cancel()
}
