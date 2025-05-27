package MyHandlers

import (
	"context"
	"github.com/ghettovoice/gosip/log"
	"github.com/redis/go-redis/v9"
)

type AsyncHandlers struct {
	RedisClient *redis.Client
	Ctx         context.Context
	Logger      log.Logger
}

type AsyncInterface interface {
	GetRedisClient() *redis.Client
	GetContext() context.Context
	GetLogger() log.Logger
}

func (h *AsyncHandlers) GetRedisClient() *redis.Client {
	return h.RedisClient
}

func (h *AsyncHandlers) GetContext() context.Context {
	return h.Ctx
}

func (h *AsyncHandlers) GetLogger() log.Logger {
	return h.Logger
}
