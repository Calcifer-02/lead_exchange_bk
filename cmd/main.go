package main

import (
	"context"
	minio "lead_exchange/internal/lib/minio/core"

	"github.com/jackc/pgx/v5/pgxpool"

	"lead_exchange/internal/app"
	"lead_exchange/internal/config"
	"lead_exchange/internal/lib/logger/handlers/slogpretty"

	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	ctx := context.Background()
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	var minioClient minio.Client
	if cfg.Minio.Enabled {
		minioClient = minio.NewMinioClient()
		err = minioClient.InitMinio(cfg.Minio)
		if err != nil {
			panic(err)
		}
	}

	application := app.New(log, cfg.GRPC.Port, pool, cfg.TokenTTL, cfg.Secret, minioClient, cfg.DisableAuth, cfg)

	go func() {
		application.GRPCServer.MustRun()
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop

	application.GRPCServer.Stop()
	log.Info("Gracefully stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
