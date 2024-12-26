package brokers

import (
	"fmt"

	"sync"

	"github.com/quickube/QScaler/api/v1alpha1"
)

var (
	BrokerRegistry = make(map[string]Broker)
	registryMutex  sync.Mutex
)

func GetBroker(namespace string, name string) (Broker, error) {
	configKey := fmt.Sprintf("%s/%s", namespace, name)

	if broker, exists := BrokerRegistry[configKey]; exists {
		return broker, nil
	}
	return nil, fmt.Errorf("broker not found for %s", configKey)
}

func NewBroker(config *v1alpha1.ScalerConfig) (Broker, error) {
	switch config.Spec.Type {
	case "redis":
		redisClient, err := updateBroker(config, func() (Broker, error) { return NewRedisClient(config) })
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Redis broker: %w", err)
		}
		return redisClient, nil
	default:
		// Check if the broker already exists
		if broker, exists := BrokerRegistry[config.Spec.Type]; exists {
			return broker, nil
		}
		return nil, fmt.Errorf("unsupported broker type: %s", config.Spec.Type)
	}
}

func updateBroker(config *v1alpha1.ScalerConfig, createFunc func() (Broker, error)) (Broker, error) {
	configKey := fmt.Sprintf("%s/%s", config.Namespace, config.Name)
	// Ensure thread-safe access to the registry
	registryMutex.Lock()
	defer registryMutex.Unlock()

	if _, exists := BrokerRegistry[configKey]; exists {
		delete(BrokerRegistry, configKey)
	}

	broker, err := createFunc()
	if err != nil {
		return nil, err
	}

	// Store the broker in the registry
	BrokerRegistry[configKey] = broker
	return broker, nil
}
