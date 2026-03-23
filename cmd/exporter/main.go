package main

import (
	"log"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/thineshsubramani/github-runner-prometheus-exporter/config"
	"github.com/thineshsubramani/github-runner-prometheus-exporter/exporter"
	"github.com/thineshsubramani/github-runner-prometheus-exporter/internal/validator"
	"github.com/thineshsubramani/github-runner-prometheus-exporter/server"
)

func main() {
	// ✅ Load YAML config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	// ✅ Fallback to default port if not defined
	if cfg.Server.ListenAddress == "" {
		cfg.Server.ListenAddress = ":9200"
		log.Println("No listen_address in config, using default 9200")
	}

	if err := validator.ValidateConfig(cfg); err != nil {
		log.Fatalf("❌ Invalid config: %v", err)
	}

	// // ✅ Path validation (Linux as default — extend for OS later)
	// runnerPath := cfg.Paths.Logs.Linux.Worker
	// if err := validator.ValidatePaths(runnerPath); err != nil {
	// 	log.Fatalf("❌ Path validation failed: %v", err)
	// }

	// ✅ Process validation
	if err := validator.ValidateRunnerProcess("Runner.Worker"); err != nil {
		log.Printf("⚠️  Warning: Runner process not active: %v", err)
	}

	// ✅ Build Prometheus exporter
	exp, err := exporter.New(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to initialize exporter: %v", err)
	}

	// ✅ Serve metrics
	handler := promhttp.HandlerFor(exp.Registry, promhttp.HandlerOpts{})
	log.Printf("🚀 Exporter starting on %s/metrics", cfg.Server.ListenAddress)
	server.Start(cfg.Server.ListenAddress, handler)
}
