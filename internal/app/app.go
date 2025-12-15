package app

import (
	"lead_exchange/internal/config"
	minio "lead_exchange/internal/lib/minio/core"
	"lead_exchange/internal/lib/ml"
	"lead_exchange/internal/repository/deal_repository"
	"lead_exchange/internal/repository/lead_repository"
	"lead_exchange/internal/repository/property_repository"
	"lead_exchange/internal/services/deal"
	"lead_exchange/internal/services/lead"
	"lead_exchange/internal/services/property"

	"github.com/jackc/pgx/v5/pgxpool"

	grpcapp "lead_exchange/internal/app/grpc"
	"lead_exchange/internal/repository/user_repository"
	"lead_exchange/internal/services/user"

	"log/slog"
	"time"
)

type App struct {
	GRPCServer *grpcapp.App
}

func New(
	log *slog.Logger, grpcPort int, pool *pgxpool.Pool,
	tokenTTL time.Duration, secret string, minioClient minio.Client, disableAuth bool, cfg *config.Config) *App {

	userRepository := user_repository.NewUserRepository(pool, log)
	leadRepository := lead_repository.NewLeadRepository(pool, log)
	dealRepository := deal_repository.NewDealRepository(pool, log)
	propertyRepository := property_repository.NewPropertyRepository(pool, log)

	// Создаём ML клиент
	mlClient := ml.NewClient(cfg.ML, log)

	userService := user.New(log, userRepository, tokenTTL, secret)
	leadService := lead.New(log, leadRepository, mlClient)
	dealService := deal.New(log, dealRepository)
	propertyService := property.New(log, propertyRepository, mlClient, leadService)

	grpcApp := grpcapp.New(log, userService, userService, minioClient, leadService, dealService, propertyService, grpcPort, secret, disableAuth)

	return &App{
		GRPCServer: grpcApp,
	}
}
