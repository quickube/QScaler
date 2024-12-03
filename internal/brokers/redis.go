package brokers

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/mitchellh/mapstructure"
	quickcubecomv1alpha1 "github.com/quickube/QScaler/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type RedisBroker struct {
	client *redis.Client
}

type RedisConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Password string `json:"password"`
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

func NewRedisClient(config *quickcubecomv1alpha1.ScalerConfig) (*RedisBroker, error) {
	redisConfig := &RedisConfig{}
	err := mapstructure.Decode(config.Spec, &redisConfig)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisConfig.Host, redisConfig.Port),
		Password: redisConfig.Password,
	})

	return &RedisBroker{
		client: client,
	}, nil
}
