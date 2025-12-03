package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tournevent/logistic/internal/server"
	"go.uber.org/zap"
)

var version = "0.0.1"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:     "logistic",
	Short:   "Delivro Logistics Bridge - Multi-carrier shipping GraphQL service",
	Version: version,
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the GraphQL server",
	RunE:  runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Initialize telemetry
	logger, err := initLogger(cfg.LogLevel)
	if err != nil {
		return err
	}
	defer logger.Sync()

	tracerShutdown, err := initTracer(ctx, cfg)
	if err != nil {
		logger.Warn("Failed to initialize tracer", zap.Error(err))
	} else {
		defer tracerShutdown(ctx)
	}

	// Initialize shipper registry with all carriers
	registry := initShipperRegistry(cfg, logger)

	logger.Info("Starting Delivro Logistics Bridge",
		zap.Int("port", cfg.Port),
		zap.String("version", cfg.Version),
	)

	// Start HTTP server
	srv := server.New(server.Config{Port: cfg.Port}, registry, logger)
	if err := srv.Run(ctx); err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}
