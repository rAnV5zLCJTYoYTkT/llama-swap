package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mostlygeek/llama-swap/proxy"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var (
		configPath  = flag.String("config", "config.yaml", "path to configuration file")
		listenAddr  = flag.String("listen", ":8080", "address to listen on")
		showVersion = flag.Bool("version", false, "print version information and exit")
		logLevel    = flag.String("log-level", "info", "log level (debug, info, warn, error)")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("llama-swap version %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := proxy.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config from %s: %v", *configPath, err)
	}

	// Create the proxy server
	server, err := proxy.New(cfg, *listenAddr, *logLevel)
	if err != nil {
		log.Fatalf("failed to create proxy server: %v", err)
	}

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Printf("received signal %s, shutting down...", sig)
		server.Stop()
	}()

	log.Printf("llama-swap %s starting on %s", version, *listenAddr)
	if err := server.Start(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
