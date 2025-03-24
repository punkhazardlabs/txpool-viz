package service

import (
	"txpool-viz/pkg"

	"github.com/redis/go-redis/v9"
)

type Service struct {
	Redis *redis.Client
	DB		any 
	Logger pkg.Logger
}
