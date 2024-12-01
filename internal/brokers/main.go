package brokers

import (
	"fmt"
	conf "github.com/quickube/QScaler/internal/config"
)

func NewBroker(cfg *conf.GlobalConfig) (Broker, error) {
	switch cfg.BrokerConfig.Provider {
	case "redis":
		redisClient, err := NewRedisClient(cfg)
		if err != nil {
			return nil, err
		}
		return redisClient, nil
	}

	return nil, fmt.Errorf("didn't find matching broker provider %s", cfg.BrokerConfig.Provider)
}
