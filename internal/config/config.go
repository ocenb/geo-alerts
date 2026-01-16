package config

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Environment      string        `env:"ENVIRONMENT" env-default:"local" validate:"oneof=local dev test prod"`
	DBConnectTimeout time.Duration `env:"DB_CONNECT_TIMEOUT" env-default:"10s" validate:"min=1s"`
	ShutdownTimeout  time.Duration `env:"SHUTDOWN_TIMEOUT" env-default:"10s" validate:"min=1s"`
	AsyncJobTimeout  time.Duration `env:"ASYNC_JOB_TIMEOUT" env-default:"10s" validate:"min=1s"`

	Log      LogConfig
	App      AppConfig
	Webhook  WebhookConfig
	Server   ServerConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	Cache    CacheConfig
	Queue    QueueConfig
}

type AppConfig struct {
	APIKey          string        `env:"API_KEY" env-required:"true"`
	StatsTimeWindow time.Duration `env:"STATS_TIME_WINDOW_MINUTES" env-default:"15m"`
}

type WebhookConfig struct {
	Port                      string        `env:"WEBHOOK_PORT" env-default:"9090" validate:"numeric"`
	URL                       string        `env:"WEBHOOK_URL" env-default:"http://localhost:9090" validate:"url"`
	RequestTimeout            time.Duration `env:"WEBHOOK_REQUEST_TIMEOUT" env-default:"10s" validate:"min=1s"`
	ClientMaxIdleConns        int           `env:"WEBHOOK_CLIENT_MAX_IDLE_CONNS" env-default:"100" validate:"min=1"`
	ClientMaxIdleConnsPerHost int           `env:"WEBHOOK_CLIENT_MAX_IDLE_CONNS_PER_HOST" env-default:"20" validate:"min=1"`
	ClientIdleConnTimeout     time.Duration `env:"WEBHOOK_CLIENT_IDLE_CONN_TIMEOUT" env-default:"90s" validate:"min=1s"`
}

type LogConfig struct {
	Level   int    `env:"LOG_LEVEL" env-default:"0" validate:"oneof=-4 0 4 8"` // -4 = Debug, 0 = Info, 4 = Warn, 8 = Error
	Handler string `env:"LOG_HANDLER" env-default:"text" validate:"oneof=text json"`
}

type ServerConfig struct {
	Port              string        `env:"PORT" env-default:"8080" validate:"numeric"`
	ReadTimeout       time.Duration `env:"READ_TIMEOUT" env-default:"10s" validate:"min=100ms"`
	WriteTimeout      time.Duration `env:"WRITE_TIMEOUT" env-default:"10s" validate:"min=100ms"`
	IdleTimeout       time.Duration `env:"IDLE_TIMEOUT" env-default:"60s" validate:"min=1s"`
	ReadHeaderTimeout time.Duration `env:"READ_HEADER_TIMEOUT" env-default:"5s" validate:"min=100ms"`
}

type PostgresConfig struct {
	Host            string        `env:"POSTGRES_HOST" env-required:"true" validate:"hostname|ip"`
	Port            string        `env:"POSTGRES_PORT" env-required:"true" validate:"numeric"`
	User            string        `env:"POSTGRES_USER" env-required:"true"`
	Password        string        `env:"POSTGRES_PASSWORD" env-required:"true"`
	Name            string        `env:"POSTGRES_DB" env-required:"true"`
	SSLMode         string        `env:"POSTGRES_SSLMODE" env-default:"disable" validate:"oneof=disable allow prefer require verify-ca verify-full"`
	MaxOpenConns    int32         `env:"POSTGRES_MAX_OPEN_CONNS" env-default:"10" validate:"min=1"`
	MaxIdleConns    int32         `env:"POSTGRES_MAX_IDLE_CONNS" env-default:"5" validate:"min=1"`
	ConnMaxLifetime time.Duration `env:"POSTGRES_CONN_MAX_LIFETIME" env-default:"1h" validate:"min=1m"`
	DSN             string
}

type RedisConfig struct {
	Host         string        `env:"REDIS_HOST" env-required:"true" validate:"hostname|ip"`
	Port         string        `env:"REDIS_PORT" env-required:"true" validate:"numeric"`
	Password     string        `env:"REDIS_PASSWORD" env-required:"true"`
	DBCache      int           `env:"REDIS_DB_CACHE" env-default:"0"`
	DBQueue      int           `env:"REDIS_DB_QUEUE" env-default:"1"`
	DialTimeout  time.Duration `env:"REDIS_DIAL_TIMEOUT" env-default:"5s" validate:"min=1s"`
	ReadTimeout  time.Duration `env:"REDIS_READ_TIMEOUT" env-default:"3s" validate:"min=100ms"`
	WriteTimeout time.Duration `env:"REDIS_WRITE_TIMEOUT" env-default:"3s" validate:"min=100ms"`
	Addr         string
}

type CacheConfig struct {
	IncidentsTTL time.Duration `env:"CACHE_INCIDENTS_TTL" env-default:"1h" validate:"min=1s"`
}

type QueueConfig struct {
	MaxRetries  int           `env:"QUEUE_MAX_RETRIES" env-default:"5" validate:"min=0"`
	Timeout     time.Duration `env:"QUEUE_TIMEOUT" env-default:"1m" validate:"min=1s"`
	Concurrency int           `env:"QUEUE_CONCURRENCY" env-default:"10" validate:"min=1"`
}

func MustLoad() *Config {
	var cfg Config

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Fatalf("cannot read config: %v", err)
	}

	validate := validator.New()
	if err := validate.Struct(&cfg); err != nil {
		log.Fatalf("config validation failed: %v", err)
	}

	if err := cfg.validateLogic(); err != nil {
		log.Fatalf("config logic error: %v", err)
	}

	hostPort := net.JoinHostPort(cfg.Postgres.Host, cfg.Postgres.Port)
	cfg.Postgres.DSN = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
		cfg.Postgres.User,
		cfg.Postgres.Password,
		hostPort,
		cfg.Postgres.Name,
		cfg.Postgres.SSLMode,
	)

	cfg.Redis.Addr = fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port)

	return &cfg
}

func (cfg *Config) validateLogic() error {
	if cfg.Postgres.MaxIdleConns > cfg.Postgres.MaxOpenConns {
		return fmt.Errorf("postgres: max_idle_conns (%d) cannot be greater than max_open_conns (%d)",
			cfg.Postgres.MaxIdleConns, cfg.Postgres.MaxOpenConns)
	}
	return nil
}
