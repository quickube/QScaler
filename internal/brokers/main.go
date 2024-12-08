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
	switch config.Spec.Type {
	case "redis":
		redisConfig := &RedisConfig{
			Host:     config.Spec.Config.Host,
			Port:     config.Spec.Config.Port,
			Password: config.Spec.Config.Password.Value,
		}
		redisClient, err := createBroker(config, func() (Broker, error) {
			return NewRedisClient(redisConfig)
		})
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

func createBroker(config *v1alpha1.ScalerConfig, createFunc func() (Broker, error)) (Broker, error) {
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

func GetBroker(namespace string, name string) (Broker, error) {
	configKey := fmt.Sprintf("%s/%s", namespace, name)

	RegistryMutex.Lock()
	defer RegistryMutex.Unlock()
	if broker, exists := BrokerRegistry[configKey]; exists {
		return broker, nil
	}
	return nil, fmt.Errorf("broker not found for %s", configKey)
}
