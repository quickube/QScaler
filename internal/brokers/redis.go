package brokers

import (
	"context"
	"errors"
	"fmt"
	"github.com/quickube/QScaler/api/v1alpha1"

	"github.com/go-redis/redis/v8"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type RedisBroker struct {
	client *redis.Client
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
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

func NewRedisClient(scaleConfig *v1alpha1.ScalerConfig) (*RedisBroker, error) {
	config := &RedisConfig{
		Host:     scaleConfig.Spec.Config.Host,
		Port:     scaleConfig.Spec.Config.Port,
		Password: scaleConfig.Spec.Config.Password.Value,
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config.Host, config.Port),
		Password: config.Password,
	})

	return &RedisBroker{
		client: redisClient,
	}, nil
}
