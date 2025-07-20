package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	NATS     NATSConfig     `yaml:"nats"`
	Redis    RedisConfig    `yaml:"redis"`
	Risk     RiskConfig     `yaml:"risk"`
	Trading  TradingConfig  `yaml:"trading"`
	Logging  LoggingConfig  `yaml:"logging"`
	Security SecurityConfig `yaml:"security"`
}

type ServerConfig struct {
	Host         string        `yaml:"host" env:"SERVER_HOST" default:"localhost"`
	Port         int           `yaml:"port" env:"SERVER_PORT" default:"8080"`
	ReadTimeout  time.Duration `yaml:"read_timeout" env:"SERVER_READ_TIMEOUT" default:"30s"`
	WriteTimeout time.Duration `yaml:"write_timeout" env:"SERVER_WRITE_TIMEOUT" default:"30s"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" env:"SERVER_IDLE_TIMEOUT" default:"60s"`
}

type DatabaseConfig struct {
	Host            string        `yaml:"host" env:"DB_HOST" default:"localhost"`
	Port            int           `yaml:"port" env:"DB_PORT" default:"5432"`
	Database        string        `yaml:"database" env:"DB_NAME,required"`
	Username        string        `yaml:"username" env:"DB_USER,required"`
	Password        string        `yaml:"password" env:"DB_PASSWORD,required"`
	SSLMode         string        `yaml:"ssl_mode" env:"DB_SSL_MODE" default:"require"`
	MaxOpenConns    int           `yaml:"max_open_conns" env:"DB_MAX_OPEN_CONNS" default:"25"`
	MaxIdleConns    int           `yaml:"max_idle_conns" env:"DB_MAX_IDLE_CONNS" default:"25"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" env:"DB_CONN_MAX_LIFETIME" default:"5m"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time" env:"DB_CONN_MAX_IDLE_TIME" default:"5m"`
}

type NATSConfig struct {
	URL               string        `yaml:"url" env:"NATS_URL" default:"nats://localhost:4222"`
	MaxReconnects     int           `yaml:"max_reconnects" env:"NATS_MAX_RECONNECTS" default:"5"`
	ReconnectWait     time.Duration `yaml:"reconnect_wait" env:"NATS_RECONNECT_WAIT" default:"2s"`
	ConnectionTimeout time.Duration `yaml:"connection_timeout" env:"NATS_CONNECTION_TIMEOUT" default:"5s"`
	DrainTimeout      time.Duration `yaml:"drain_timeout" env:"NATS_DRAIN_TIMEOUT" default:"5s"`
}

type RedisConfig struct {
	Addr         string        `yaml:"addr" env:"REDIS_ADDR" default:"localhost:6379"`
	Password     string        `yaml:"password" env:"REDIS_PASSWORD"`
	DB           int           `yaml:"db" env:"REDIS_DB" default:"0"`
	DialTimeout  time.Duration `yaml:"dial_timeout" env:"REDIS_DIAL_TIMEOUT" default:"5s"`
	ReadTimeout  time.Duration `yaml:"read_timeout" env:"REDIS_READ_TIMEOUT" default:"3s"`
	WriteTimeout time.Duration `yaml:"write_timeout" env:"REDIS_WRITE_TIMEOUT" default:"3s"`
	PoolSize     int           `yaml:"pool_size" env:"REDIS_POOL_SIZE" default:"10"`
}

type RiskConfig struct {
	MaxPositionSize    float64 `yaml:"max_position_size" env:"RISK_MAX_POSITION_SIZE" default:"0.1"`
	MaxConcentration   float64 `yaml:"max_concentration" env:"RISK_MAX_CONCENTRATION" default:"0.2"`
	MaxLeverage        float64 `yaml:"max_leverage" env:"RISK_MAX_LEVERAGE" default:"2.0"`
	MaxDailyLoss       float64 `yaml:"max_daily_loss" env:"RISK_MAX_DAILY_LOSS" default:"0.05"`
	MaxVaR             float64 `yaml:"max_var" env:"RISK_MAX_VAR" default:"0.02"`
	VaRConfidenceLevel float64 `yaml:"var_confidence_level" env:"RISK_VAR_CONFIDENCE" default:"0.95"`
}

type TradingConfig struct {
	DefaultSlippage    float64       `yaml:"default_slippage" env:"TRADING_DEFAULT_SLIPPAGE" default:"0.001"`
	MaxOrderSize       float64       `yaml:"max_order_size" env:"TRADING_MAX_ORDER_SIZE" default:"1000000"`
	OrderTimeout       time.Duration `yaml:"order_timeout" env:"TRADING_ORDER_TIMEOUT" default:"30s"`
	MarketDataTimeout  time.Duration `yaml:"market_data_timeout" env:"TRADING_MARKET_DATA_TIMEOUT" default:"5s"`
	CommissionRate     float64       `yaml:"commission_rate" env:"TRADING_COMMISSION_RATE" default:"0.001"`
}

type LoggingConfig struct {
	Level      string `yaml:"level" env:"LOG_LEVEL" default:"info"`
	Format     string `yaml:"format" env:"LOG_FORMAT" default:"json"`
	Output     string `yaml:"output" env:"LOG_OUTPUT" default:"stdout"`
	Filename   string `yaml:"filename" env:"LOG_FILENAME"`
	MaxSize    int    `yaml:"max_size" env:"LOG_MAX_SIZE" default:"100"`
	MaxBackups int    `yaml:"max_backups" env:"LOG_MAX_BACKUPS" default:"3"`
	MaxAge     int    `yaml:"max_age" env:"LOG_MAX_AGE" default:"7"`
	Compress   bool   `yaml:"compress" env:"LOG_COMPRESS" default:"true"`
}

type SecurityConfig struct {
	JWTSecret           string        `yaml:"jwt_secret" env:"JWT_SECRET,required"`
	APIKeyRotationDays  int           `yaml:"api_key_rotation_days" env:"API_KEY_ROTATION_DAYS" default:"30"`
	RateLimitPerMinute  int           `yaml:"rate_limit_per_minute" env:"RATE_LIMIT_PER_MINUTE" default:"1000"`
	SessionTimeout      time.Duration `yaml:"session_timeout" env:"SESSION_TIMEOUT" default:"24h"`
	TLSEnabled          bool          `yaml:"tls_enabled" env:"TLS_ENABLED" default:"true"`
	CertFile            string        `yaml:"cert_file" env:"TLS_CERT_FILE"`
	KeyFile             string        `yaml:"key_file" env:"TLS_KEY_FILE"`
}

func Load() (*Config, error) {
	config := &Config{}

	if err := loadFromEnv(config); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := validate(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

func loadFromEnv(config *Config) error {
	config.Server = ServerConfig{
		Host:         getEnvOrDefault("SERVER_HOST", "localhost"),
		Port:         getEnvIntOrDefault("SERVER_PORT", 8080),
		ReadTimeout:  getEnvDurationOrDefault("SERVER_READ_TIMEOUT", 30*time.Second),
		WriteTimeout: getEnvDurationOrDefault("SERVER_WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:  getEnvDurationOrDefault("SERVER_IDLE_TIMEOUT", 60*time.Second),
	}

	config.Database = DatabaseConfig{
		Host:            getEnvOrDefault("DB_HOST", "localhost"),
		Port:            getEnvIntOrDefault("DB_PORT", 5432),
		Database:        os.Getenv("DB_NAME"),
		Username:        os.Getenv("DB_USER"),
		Password:        os.Getenv("DB_PASSWORD"),
		SSLMode:         getEnvOrDefault("DB_SSL_MODE", "require"),
		MaxOpenConns:    getEnvIntOrDefault("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvIntOrDefault("DB_MAX_IDLE_CONNS", 25),
		ConnMaxLifetime: getEnvDurationOrDefault("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		ConnMaxIdleTime: getEnvDurationOrDefault("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
	}

	config.NATS = NATSConfig{
		URL:               getEnvOrDefault("NATS_URL", "nats://localhost:4222"),
		MaxReconnects:     getEnvIntOrDefault("NATS_MAX_RECONNECTS", 5),
		ReconnectWait:     getEnvDurationOrDefault("NATS_RECONNECT_WAIT", 2*time.Second),
		ConnectionTimeout: getEnvDurationOrDefault("NATS_CONNECTION_TIMEOUT", 5*time.Second),
		DrainTimeout:      getEnvDurationOrDefault("NATS_DRAIN_TIMEOUT", 5*time.Second),
	}

	config.Redis = RedisConfig{
		Addr:         getEnvOrDefault("REDIS_ADDR", "localhost:6379"),
		Password:     os.Getenv("REDIS_PASSWORD"),
		DB:           getEnvIntOrDefault("REDIS_DB", 0),
		DialTimeout:  getEnvDurationOrDefault("REDIS_DIAL_TIMEOUT", 5*time.Second),
		ReadTimeout:  getEnvDurationOrDefault("REDIS_READ_TIMEOUT", 3*time.Second),
		WriteTimeout: getEnvDurationOrDefault("REDIS_WRITE_TIMEOUT", 3*time.Second),
		PoolSize:     getEnvIntOrDefault("REDIS_POOL_SIZE", 10),
	}

	config.Risk = RiskConfig{
		MaxPositionSize:    getEnvFloatOrDefault("RISK_MAX_POSITION_SIZE", 0.1),
		MaxConcentration:   getEnvFloatOrDefault("RISK_MAX_CONCENTRATION", 0.2),
		MaxLeverage:        getEnvFloatOrDefault("RISK_MAX_LEVERAGE", 2.0),
		MaxDailyLoss:       getEnvFloatOrDefault("RISK_MAX_DAILY_LOSS", 0.05),
		MaxVaR:             getEnvFloatOrDefault("RISK_MAX_VAR", 0.02),
		VaRConfidenceLevel: getEnvFloatOrDefault("RISK_VAR_CONFIDENCE", 0.95),
	}

	config.Trading = TradingConfig{
		DefaultSlippage:   getEnvFloatOrDefault("TRADING_DEFAULT_SLIPPAGE", 0.001),
		MaxOrderSize:      getEnvFloatOrDefault("TRADING_MAX_ORDER_SIZE", 1000000),
		OrderTimeout:      getEnvDurationOrDefault("TRADING_ORDER_TIMEOUT", 30*time.Second),
		MarketDataTimeout: getEnvDurationOrDefault("TRADING_MARKET_DATA_TIMEOUT", 5*time.Second),
		CommissionRate:    getEnvFloatOrDefault("TRADING_COMMISSION_RATE", 0.001),
	}

	config.Logging = LoggingConfig{
		Level:      getEnvOrDefault("LOG_LEVEL", "info"),
		Format:     getEnvOrDefault("LOG_FORMAT", "json"),
		Output:     getEnvOrDefault("LOG_OUTPUT", "stdout"),
		Filename:   os.Getenv("LOG_FILENAME"),
		MaxSize:    getEnvIntOrDefault("LOG_MAX_SIZE", 100),
		MaxBackups: getEnvIntOrDefault("LOG_MAX_BACKUPS", 3),
		MaxAge:     getEnvIntOrDefault("LOG_MAX_AGE", 7),
		Compress:   getEnvBoolOrDefault("LOG_COMPRESS", true),
	}

	config.Security = SecurityConfig{
		JWTSecret:          os.Getenv("JWT_SECRET"),
		APIKeyRotationDays: getEnvIntOrDefault("API_KEY_ROTATION_DAYS", 30),
		RateLimitPerMinute: getEnvIntOrDefault("RATE_LIMIT_PER_MINUTE", 1000),
		SessionTimeout:     getEnvDurationOrDefault("SESSION_TIMEOUT", 24*time.Hour),
		TLSEnabled:         getEnvBoolOrDefault("TLS_ENABLED", true),
		CertFile:           os.Getenv("TLS_CERT_FILE"),
		KeyFile:            os.Getenv("TLS_KEY_FILE"),
	}

	return nil
}

func validate(config *Config) error {
	if config.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}
	if config.Database.Username == "" {
		return fmt.Errorf("database username is required")
	}
	if config.Database.Password == "" {
		return fmt.Errorf("database password is required")
	}
	if config.Security.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required")
	}
	if len(config.Security.JWTSecret) < 32 {
		return fmt.Errorf("JWT secret must be at least 32 characters")
	}

	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloatOrDefault(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}