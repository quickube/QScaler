package brokers

import "context"

type Broker interface {
	GetQueueLength(ctx *context.Context, topic string) (int, error)
	GetDeathQueue(topic string) string
	IsConnected(ctx *context.Context) (bool, error)
}
