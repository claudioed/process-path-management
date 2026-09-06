// Command pathmgmt is the composition root for the Process Path
// Management service: it wires config from the environment to adapters,
// use cases, and the HTTP router, then serves it.
//
// Unlike most of this fleet's services, this one has no inbound Kafka
// consumer — it is the SOURCE of the process-path published language,
// not a consumer of anyone else's events. Its only Kafka role is the
// outbound publisher.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"

	inboundhttp "github.com/claudioed/process-path-management/internal/adapters/inbound/http"
	"github.com/claudioed/process-path-management/internal/adapters/outbound/events"
	outboundkafka "github.com/claudioed/process-path-management/internal/adapters/outbound/kafka"
	"github.com/claudioed/process-path-management/internal/adapters/outbound/memory"
	"github.com/claudioed/process-path-management/internal/adapters/outbound/postgres"
	"github.com/claudioed/process-path-management/internal/adapters/outbound/telemetry"
	"github.com/claudioed/process-path-management/internal/application/ports"
	"github.com/claudioed/process-path-management/internal/application/usecases"
)

// version is the service version reported as the OTel `service.version`
// resource attribute. Overridable at build time
// (-ldflags "-X main.version=1.2.3"), else SERVICE_VERSION, else "dev".
var version = ""

func main() {
	if err := run(); err != nil {
		slog.Error("service exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	logger := newLogger(getenv("LOG_LEVEL", "info"))
	slog.SetDefault(logger)

	ctx := context.Background()

	serviceName := getenv("OTEL_SERVICE_NAME", inboundhttp.DefaultServiceName)
	otlpEndpoint := getenv("OTEL_EXPORTER_OTLP_ENDPOINT", telemetry.DefaultOTLPEndpoint)
	shutdownTelemetry, err := telemetry.Setup(ctx, serviceName, serviceVersion(), otlpEndpoint)
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTelemetry(shutdownCtx); err != nil {
			logger.Warn("telemetry shutdown did not flush cleanly", "error", err)
		}
	}()
	logger.Info("telemetry configured",
		"service_name", serviceName,
		"service_version", serviceVersion(),
		"environment", telemetry.Environment(),
		"otlp_endpoint", otlpEndpoint,
	)

	httpAddr := getenv("HTTP_ADDR", ":8080")
	databaseURL := os.Getenv("DATABASE_URL")
	migrationsPath := getenv("MIGRATIONS_PATH", "migrations")

	repo, closeRepo, err := buildRepo(ctx, databaseURL, migrationsPath, logger)
	if err != nil {
		return err
	}
	defer closeRepo()

	publisher, closePublisher := buildEventPublisher(logger)
	defer closePublisher()
	clock := memory.SystemClock{}

	pathMetrics, err := telemetry.NewPathMetrics()
	if err != nil {
		return err
	}

	server := &inboundhttp.Server{
		DefinePath:     &usecases.DefinePath{Repo: repo, Publisher: publisher, Clock: clock, Metrics: pathMetrics},
		RevisePath:     &usecases.RevisePath{Repo: repo, Publisher: publisher, Clock: clock},
		DeactivatePath: &usecases.DeactivatePath{Repo: repo, Publisher: publisher, Clock: clock},
		GetPath:        &usecases.GetPath{Repo: repo},
		ListPaths:      &usecases.ListPaths{Repo: repo},
	}

	httpServer := &http.Server{
		Addr:              httpAddr,
		Handler:           inboundhttp.NewRouter(server, logger, serviceName),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "addr", httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stopCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		return err
	case <-stopCtx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return httpServer.Shutdown(shutdownCtx)
}

// newLogger builds the process-wide structured logger, wrapped so any
// *Context log call made while a span is active carries trace_id/span_id.
func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(telemetry.NewTraceHandler(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}),
	))
}

// serviceVersion resolves service.version: build-time ldflags first, then
// SERVICE_VERSION, then "dev".
func serviceVersion() string {
	if version != "" {
		return version
	}
	return getenv("SERVICE_VERSION", "dev")
}

// buildRepo wires the outbound ProcessPathRepo. With no DATABASE_URL set,
// the service runs fully functional against an in-memory repo (no
// Postgres required for local dev / smoke tests) — same fallback
// convention as every other service in this fleet.
func buildRepo(ctx context.Context, databaseURL, migrationsPath string, logger *slog.Logger) (ports.ProcessPathRepo, func(), error) {
	if databaseURL == "" {
		logger.Info("DATABASE_URL not set, using in-memory ProcessPathRepo")
		return memory.NewProcessPathRepo(), func() {}, nil
	}

	if err := postgres.RunMigrations(databaseURL, migrationsPath); err != nil {
		return nil, nil, err
	}
	pool, err := postgres.NewPool(ctx, databaseURL)
	if err != nil {
		return nil, nil, err
	}
	return postgres.NewProcessPathRepo(pool), pool.Close, nil
}

// buildEventPublisher wires the outbound event publisher, returning it
// and a close function.
//
// The default is the log publisher, so a local dev run with no Kafka is
// still fully functional. Setting EVENT_PUBLISHER=kafka publishes onto
// warehouse.process-path-management.events, which is what
// fulfillment-execution, wes-work-planning, and workforce-management
// consume to keep their own local path caches current.
func buildEventPublisher(logger *slog.Logger) (ports.EventPublisher, func()) {
	if !strings.EqualFold(getenv("EVENT_PUBLISHER", "log"), "kafka") {
		return events.NewLogPublisher(logger), func() {}
	}

	brokers := strings.Split(getenv("KAFKA_BROKERS", "localhost:9092"), ",")
	publisher := outboundkafka.NewPublisher(brokers, uuid.NewString)
	logger.Info("kafka event publishing enabled", "brokers", brokers, "topic", outboundkafka.Topic)
	return publisher, func() {
		if err := publisher.Close(); err != nil {
			logger.Error("error closing kafka publisher", "error", err)
		}
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
