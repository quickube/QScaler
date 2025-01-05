package brokers

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/mitchellh/mapstructure"
	"github.com/quickube/QScaler/api/v1alpha1"
	"github.com/quickube/QScaler/internal/secret_manager"
)

type RedisBroker struct {
	client *redis.Client
}

type RedisConfig struct {
	Host     string                 `yaml:"host"`
	Port     string                 `yaml:"port"`
	Password v1alpha1.ValueOrSecret `yaml:"password"`
}

func (r *RedisBroker) GetQueueLength(ctx *context.Context, topic string) (int, error) {
	taskQueueLength, err := r.client.LLen(*ctx, topic).Result()
	if err != nil {
		return -1, err
	}
	return int(taskQueueLength), nil
}

func (r *RedisBroker) IsConnected(ctx *context.Context) (bool, error) {
	status := r.client.Ping(*ctx)
	return status.Err() == nil, status.Err()
}

func NewRedisClient(config *v1alpha1.ScalerConfig) (*RedisBroker, error) {
	redisConfig := &RedisConfig{}
	err := mapstructure.Decode(config.Spec.Config.RedisConfig, &redisConfig)
	if err != nil {
		return nil, err
	}

	secretManager, err := secret_manager.NewClient()
	if err != nil {
		return nil, err
	}

	password, err := secretManager.Get(redisConfig.Password)
	if err != nil {
		return nil, err
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisConfig.Host, redisConfig.Port),
		Password: password,
	})

	return &RedisBroker{
		client: redisClient,
	}, nil
}
