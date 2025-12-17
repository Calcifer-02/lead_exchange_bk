package config

import (
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env         string `env:"ENV" env-default:"local"`
	DatabaseURL string `env:"DATABASE_URL" env-required:"true"`
	GRPC        GRPCConfig
	TokenTTL    time.Duration `env:"TOKEN_TTL" env-default:"1h"`
	Secret      string        `env:"SECRET" env-required:"true"`
	DisableAuth bool          `env:"DISABLE_AUTH" env-default:"false"`
	Minio       MinioConfig
	ML          MLConfig
	Reranker    RerankerConfig
	LLM         LLMConfig
	Vision      VisionConfig
	Search      SearchConfig
}

type GRPCConfig struct {
	Port    int           `env:"GRPC_PORT" env-default:"44044"`
	Timeout time.Duration `env:"GRPC_TIMEOUT" env-default:"10h"`
}

type MinioConfig struct {
	Enabled           bool   `env:"MINIO_ENABLE" env-default:"false"`
	Port              int    `env:"MINIO_PORT" env-default:"9000"`
	MinioEndpoint     string `env:"MINIO_ENDPOINT"`
	BucketName        string `env:"MINIO_BUCKET"`
	MinioRootUser     string `env:"MINIO_USER"`
	MinioRootPassword string `env:"MINIO_PASSWORD"`
	MinioUseSSL       bool   `env:"MINIO_USE_SSL"`
}

type MLConfig struct {
	Enabled  bool   `env:"ML_ENABLE" env-default:"true"`
	BaseURL  string `env:"ML_BASE_URL" env-default:"https://calcifer0323-matching.hf.space"`
	Timeout  time.Duration `env:"ML_TIMEOUT" env-default:"30s"`
}

// RerankerConfig — конфигурация для Reranker API (Jina AI, Cohere и др.).
type RerankerConfig struct {
	Enabled bool          `env:"RERANKER_ENABLE" env-default:"false"`
	BaseURL string        `env:"RERANKER_BASE_URL" env-default:"https://api.jina.ai/v1"`
	APIKey  string        `env:"RERANKER_API_KEY"`
	Model   string        `env:"RERANKER_MODEL" env-default:"jina-reranker-v2-base-multilingual"`
	Timeout time.Duration `env:"RERANKER_TIMEOUT" env-default:"30s"`
	TopN    int           `env:"RERANKER_TOP_N" env-default:"10"`
}

// LLMConfig — конфигурация для LLM API (OpenAI, Azure OpenAI и др.).
type LLMConfig struct {
	Enabled bool          `env:"LLM_ENABLE" env-default:"false"`
	BaseURL string        `env:"LLM_BASE_URL" env-default:"https://api.openai.com/v1"`
	APIKey  string        `env:"LLM_API_KEY"`
	Model   string        `env:"LLM_MODEL" env-default:"gpt-4o-mini"`
	Timeout time.Duration `env:"LLM_TIMEOUT" env-default:"60s"`
}

// VisionConfig — конфигурация для Computer Vision API.
type VisionConfig struct {
	Enabled bool          `env:"VISION_ENABLE" env-default:"false"`
	BaseURL string        `env:"VISION_BASE_URL"`
	APIKey  string        `env:"VISION_API_KEY"`
	Timeout time.Duration `env:"VISION_TIMEOUT" env-default:"30s"`
}

// SearchConfig — конфигурация для гибридного поиска.
type SearchConfig struct {
	// HybridSearchEnabled включает комбинирование векторного и полнотекстового поиска
	HybridSearchEnabled bool `env:"HYBRID_SEARCH_ENABLE" env-default:"true"`
	// VectorWeight — вес векторного поиска (0-1)
	VectorWeight float64 `env:"SEARCH_VECTOR_WEIGHT" env-default:"0.7"`
	// FulltextWeight — вес полнотекстового поиска (0-1)
	FulltextWeight float64 `env:"SEARCH_FULLTEXT_WEIGHT" env-default:"0.3"`
	// UseReranker — использовать reranker для финального ранжирования
	UseReranker bool `env:"SEARCH_USE_RERANKER" env-default:"false"`
	// RerankerCandidates — количество кандидатов для передачи в reranker
	RerankerCandidates int `env:"SEARCH_RERANKER_CANDIDATES" env-default:"50"`
	// DynamicWeightsEnabled — использовать динамическое определение весов через LLM
	DynamicWeightsEnabled bool `env:"DYNAMIC_WEIGHTS_ENABLE" env-default:"false"`
}

func MustLoad() *Config {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		panic("cannot read config from environment: " + err.Error())
	}
	return &cfg
}
