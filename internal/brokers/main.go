package brokers

import (
	"fmt"
	"sync"

	"github.com/quickube/QScaler/api/v1alpha1"
)

var (
	BrokerRegistry = make(map[string]Broker)
	RegistryMutex  sync.Mutex
)

func NewBroker(config *v1alpha1.ScalerConfig) (Broker, error) {
	switch config.Type {
	case "redis":
		redisClient, err := getBroker(config, func() (Broker, error) {
			return NewRedisClient(config)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Redis broker: %w", err)
		}
		return redisClient, nil
	default:
		return nil, fmt.Errorf("unsupported broker type: %s", config.Type)
	}
}

func getBroker(config *v1alpha1.ScalerConfig, createFunc func() (Broker, error)) (Broker, error) {
	configKey := fmt.Sprintf("%s/%s", config.Namespace, config.Name)
	// Ensure thread-safe access to the registry
	RegistryMutex.Lock()
	defer RegistryMutex.Unlock()

	// Check if the broker already exists
	if broker, exists := BrokerRegistry[configKey]; exists {
		return broker, nil
	}

	// Create a new broker if it doesn't exist
	broker, err := createFunc()
	if err != nil {
		return nil, err
	}

	// Store the broker in the registry
	BrokerRegistry[configKey] = broker
	return broker, nil
}
