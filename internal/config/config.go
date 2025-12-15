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

func MustLoad() *Config {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		panic("cannot read config from environment: " + err.Error())
	}
	return &cfg
}
