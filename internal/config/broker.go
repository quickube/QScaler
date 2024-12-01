package conf

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type BrokerConfig struct {
	Provider    string `envconfig:"BROKER_PROVIDER" required:"true"`
	Address     string `envconfig:"BROKER_ADDRESS" required:"true"`
	Credentials string `envconfig:"BROKER_CREDENTIALS" required:"true"`
}

func (cfg *BrokerConfig) BrokerConfigLoad() error {
	err := envconfig.Process("", cfg)
	if err != nil {
		return fmt.Errorf("failed to load the Git provider configuration, error: %v", err)
	}

	return nil
}
