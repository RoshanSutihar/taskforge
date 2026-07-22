package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	DatabaseURL       string `mapstructure:"DATABASE_URL"`
	RedisURL          string `mapstructure:"REDIS_URL"`
	WorkerConcurrency int    `mapstructure:"WORKER_CONCURRENCY"`
	HeartbeatInterval int    `mapstructure:"HEARTBEAT_INTERVAL"`
	HeartbeatTimeout  int    `mapstructure:"HEARTBEAT_TIMEOUT"`
	ShutdownTimeout   int    `mapstructure:"SHUTDOWN_TIMEOUT"`
	SchedulerInterval int    `mapstructure:"SCHEDULER_INTERVAL"`
	ReaperInterval    int    `mapstructure:"REAPER_INTERVAL"`
	OrchestratorPort  string `mapstructure:"ORCHESTRATOR_PORT"`
	WorkerID          string `mapstructure:"WORKER_ID"`
	Environment       string `mapstructure:"ENVIRONMENT"`
}

func Load() (*Config, error) {
	viper.SetDefault("WORKER_CONCURRENCY", 10)
	viper.SetDefault("HEARTBEAT_INTERVAL", 5)
	viper.SetDefault("HEARTBEAT_TIMEOUT", 15)
	viper.SetDefault("SHUTDOWN_TIMEOUT", 30)
	viper.SetDefault("SCHEDULER_INTERVAL", 1)
	viper.SetDefault("REAPER_INTERVAL", 10)
	viper.SetDefault("ORCHESTRATOR_PORT", ":8080")
	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("REDIS_URL", "redis://redis:6379/0")

	viper.AutomaticEnv()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
