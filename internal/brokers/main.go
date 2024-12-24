package brokers

import (
	"fmt"
	"sync"

	"github.com/quickube/QScaler/api/v1alpha1"
)

var (
	brokerRegistry = make(map[string]Broker)
	registryMutex  sync.Mutex
)

func UpdateBroker(config *v1alpha1.ScalerConfig) (Broker, error) {
	configKey := fmt.Sprintf("%s/%s", config.Namespace, config.Name)
	// Ensure thread-safe access to the registry
	registryMutex.Lock()
	defer registryMutex.Unlock()

	if _, exists := brokerRegistry[configKey]; exists {
		delete(brokerRegistry, configKey)
	}
	broker, err := NewBroker(config)
	if err != nil {
		return nil, err
	}

	return broker, nil
}

func GetBroker(namespace string, name string) (Broker, error) {
	configKey := fmt.Sprintf("%s/%s", namespace, name)

	if broker, exists := brokerRegistry[configKey]; exists {
		return broker, nil
	}
	return nil, fmt.Errorf("broker not found for %s", configKey)
}

func NewBroker(config *v1alpha1.ScalerConfig) (Broker, error) {
	switch config.Spec.Type {
	case "redis":
		redisClient, err := getBroker(config, func() (Broker, error) { return NewRedisClient(config) })
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Redis broker: %w", err)
		}
		return redisClient, nil
	default:
		// Check if the broker already exists
		if broker, exists := brokerRegistry[config.Spec.Type]; exists {
			return broker, nil
		}
		return nil, fmt.Errorf("unsupported broker type: %s", config.Spec.Type)
	}
}

func getBroker(config *v1alpha1.ScalerConfig, createFunc func() (Broker, error)) (Broker, error) {
	configKey := fmt.Sprintf("%s/%s", config.Namespace, config.Name)
	// Ensure thread-safe access to the registry
	registryMutex.Lock()
	defer registryMutex.Unlock()

	// Check if the broker already exists
	if broker, exists := brokerRegistry[configKey]; exists {
		return broker, nil
	}

	// Create a new broker if it doesn't exist
	broker, err := createFunc()
	if err != nil {
		return nil, err
	}

	// Store the broker in the registry
	brokerRegistry[configKey] = broker
	return broker, nil
}
