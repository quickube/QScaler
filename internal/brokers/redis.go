package brokers

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	conf "github.com/quickube/QScaler/internal/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type RedisBroker struct {
	client *redis.Client
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

func (r *RedisBroker) IsConnected(ctx *context.Context) bool {
	status := r.client.Ping(*ctx)
	return status.Err() == nil
}

func (r *RedisBroker) GetDeathQueue(topic string) string {
	return fmt.Sprintf("death-%s", topic)
}

func NewRedisClient(cfg *conf.GlobalConfig) (*RedisBroker, error) {

	return &RedisBroker{
		client: redis.NewClient(&redis.Options{
			Addr:     cfg.Address,
			Password: cfg.Credentials,
		}),
	}, nil
}
