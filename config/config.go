package config

import (
	"fmt"
	"os"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/redis/go-redis/v9"
	"gopkg.in/yaml.v3"
)

type Endpoint struct {
	Name        string
	Url         string
	AuthHeaders map[string]string `yaml:"auth_headers"`
}

type UserConfig struct {
	Endpoints []Endpoint               `yaml:"endpoints"`
	Polling   map[string]time.Duration `yaml:"polling"`
	Filters   map[string]string        `yaml:"filters"`
}

type Config struct {
	UserCfg     *UserConfig
	RedisClient *redis.Client
	Db          string
}

func Load() (*Config, error) {
	userConfig := &UserConfig{}
	cfgData, err := os.ReadFile("config.yaml")

	if err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	err = yaml.Unmarshal(cfgData, &userConfig)

	if err != nil {
		throwErr := fmt.Errorf("Error parsing config file: %v", err)
		panic(throwErr)
	}

	// Initialize redis client
	redisUrl := os.Getenv("REDIS_URL")

	if redisUrl == "" {
		return nil, fmt.Errorf("Error reading REDIS_URL: %v", err)
	}

	redisOptions, err := redis.ParseURL(redisUrl)

	if err != nil {
		return nil, fmt.Errorf("Error parsing REDIS_URL: %v", err)
	}

	redisClient := redis.NewClient(redisOptions)

	// Initialize Postgres connection
	conn := os.Getenv("POSTGRES_URL")

	if conn == "" {
		return nil, fmt.Errorf("Error reading POSTGRES_URL: %v", err)
	}

	return &Config{
		UserCfg:     userConfig,
		RedisClient: redisClient,
		Db:          "db",
	}, nil
}
