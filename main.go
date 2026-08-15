package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"origin-ops/internal/api"
	"origin-ops/internal/config"
	"origin-ops/internal/inventory"
	"origin-ops/internal/metrics"
	"origin-ops/internal/store"
)

//go:embed index.html styles.css app.js
var staticFiles embed.FS

func main() {
	configPath := flag.String("config", "", "path to an optional JSON configuration file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("load configuration", "error", err)
		os.Exit(1)
	}

	metricStore, err := store.New(cfg.MetricsDir)
	if err != nil {
		slog.Error("open metric store", "error", err)
		os.Exit(1)
	}

	collector, err := metrics.NewCollector()
	if err != nil {
		slog.Error("initialize metric collector", "error", err)
		os.Exit(1)
	}
	applicationInventory := inventory.New(cfg.Applications)

	rootContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sampler := metrics.NewSampler(collector, metricStore, cfg.SamplingInterval())
	go sampler.Run(rootContext)

	assets, err := fs.Sub(staticFiles, ".")
	if err != nil {
		slog.Error("open embedded assets", "error", err)
		os.Exit(1)
	}

	handler := api.NewHandler(api.Dependencies{
		Assets:           assets,
		Sampler:          sampler,
		Store:            metricStore,
		SamplingInterval: cfg.SamplingInterval(),
		MaxQueryDays:     cfg.MaxQueryDays,
		MaxChartPoints:   cfg.MaxChartPoints,
		Inventory:        applicationInventory,
	})
	server := &http.Server{
		Addr:              cfg.ListenAddress,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}

	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("origin-ops listening", "address", cfg.ListenAddress)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-rootContext.Done():
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server stopped", "error", err)
			os.Exit(1)
		}
		return
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown HTTP server: %v\n", err)
		os.Exit(1)
	}
}
