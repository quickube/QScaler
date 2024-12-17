package brokers

import (
	"context"
	"errors"
	"fmt"
	"github.com/quickube/QScaler/api/v1alpha1"
	"github.com/quickube/QScaler/internal/secret_manager"

	"github.com/go-redis/redis/v8"
	"github.com/mitchellh/mapstructure"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type RedisBroker struct {
	client *redis.Client
}

type RedisConfig struct {
	Host     string                 `yaml:"host"`
	Port     string                 `yaml:"port"`
	Password v1alpha1.ValueOrSecret `yaml:"password"`
}

func (r *RedisBroker) KillQueue(ctx *context.Context, topic string) error {
	status := r.client.LPush(*ctx, r.GetDeathQueue(topic), "{'kill': 'true'}")
	if status.String() == "error" {
		return errors.New(status.String())
	}
	log.Log.Info(fmt.Sprintf("published message to death queue: %s", status))

	return nil
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

func (r *RedisBroker) GetDeathQueue(topic string) string {
	return fmt.Sprintf("death-%s", topic)
}

func NewRedisClient(config *v1alpha1.ScalerConfig) (*RedisBroker, error) {
	redisConfig := &RedisConfig{}
	err := mapstructure.Decode(config.Spec, &redisConfig)
	if err != nil {
		return nil, err
	}

	secretManager := secret_manager.NewClient()
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
