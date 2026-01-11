package app

import (
	"lead_exchange/internal/config"
	minio "lead_exchange/internal/lib/minio/core"
	"lead_exchange/internal/lib/ml"
	"lead_exchange/internal/lib/llm"
	"lead_exchange/internal/lib/metrics"
	"lead_exchange/internal/lib/reranker"
	"lead_exchange/internal/lib/vision"
	"lead_exchange/internal/repository/deal_repository"
	"lead_exchange/internal/repository/lead_repository"
	"lead_exchange/internal/repository/property_repository"
	"lead_exchange/internal/services/clarification"
	"lead_exchange/internal/services/deal"
	"lead_exchange/internal/services/lead"
	"lead_exchange/internal/services/property"
	"lead_exchange/internal/services/weights"

	"github.com/jackc/pgx/v5/pgxpool"

	grpcapp "lead_exchange/internal/app/grpc"
	"lead_exchange/internal/repository/user_repository"
	"lead_exchange/internal/services/user"

	"log/slog"
	"time"
)

type App struct {
	GRPCServer *grpcapp.App
	// AI-related clients (exported for external access)
	LLMClient      llm.Client
	RerankerClient reranker.Client
	VisionClient   vision.Client
	AIMetrics      *metrics.AIMetrics
}

func New(
	log *slog.Logger, grpcPort int, pool *pgxpool.Pool,
	tokenTTL time.Duration, secret string, minioClient minio.Client, disableAuth bool, cfg *config.Config) *App {

	userRepository := user_repository.NewUserRepository(pool, log)
	leadRepository := lead_repository.NewLeadRepository(pool, log)
	dealRepository := deal_repository.NewDealRepository(pool, log)
	propertyRepository := property_repository.NewPropertyRepository(pool, log)

	// Создаём ML клиент (embeddings)
	mlClient := ml.NewClient(cfg.ML, log)

	// Создаём AI-клиенты
	llmClient := llm.NewClient(cfg.LLM, log)
	rerankerClient := reranker.NewClient(cfg.Reranker, log)
	visionClient := vision.NewClient(cfg.Vision, log)

	// Создаём AI-метрики
	aiMetrics := metrics.GetAIMetrics(log)

	// Логируем статус AI-сервисов
	log.Info("AI services initialized",
		slog.Bool("llm_enabled", llmClient.IsEnabled()),
		slog.Bool("reranker_enabled", rerankerClient.IsEnabled()),
		slog.Bool("vision_enabled", visionClient.IsEnabled()),
		slog.Bool("dynamic_weights_enabled", cfg.Search.DynamicWeightsEnabled),
		slog.Bool("hybrid_search_enabled", cfg.Search.HybridSearchEnabled),
	)

	// Создаём анализатор весов (использует LLM для динамического определения весов)
	weightsAnalyzer := weights.NewAnalyzer(log, llmClient, cfg.Search)

	// Создаём агента для уточняющих вопросов (использует LLM)
	clarificationAgent := clarification.NewAgent(log, llmClient, weightsAnalyzer)

	userService := user.New(log, userRepository, tokenTTL, secret)
	leadService := lead.New(log, leadRepository, mlClient)
	dealService := deal.New(log, dealRepository)

	// Создаём property service с поддержкой расширенного поиска
	propertyService := property.NewWithAdvancedSearch(
		log,
		propertyRepository,
		mlClient,
		rerankerClient,
		weightsAnalyzer,
		leadService,
		cfg.Search,
	)

	// Создаём gRPC приложение с AI-клиентами
	grpcApp := grpcapp.NewWithAI(
		log,
		userService,
		userService,
		minioClient,
		leadService,
		dealService,
		propertyService,
		clarificationAgent,
		weightsAnalyzer,
		llmClient,
		visionClient,
		grpcPort,
		secret,
		disableAuth,
	)

	return &App{
		GRPCServer:     grpcApp,
		LLMClient:      llmClient,
		RerankerClient: rerankerClient,
		VisionClient:   visionClient,
		AIMetrics:      aiMetrics,
	}
}
