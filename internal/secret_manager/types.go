package secret_manager

import (
	"github.com/quickube/QScaler/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

type SecretManager interface {
	Add(configName types.NamespacedName, secret v1alpha1.ValueOrSecret) error
	Delete(configName types.NamespacedName, secret v1alpha1.ValueOrSecret) error
	Get(configName types.NamespacedName, secret v1alpha1.ValueOrSecret) (string, error)
	Sync(configName types.NamespacedName) error
	ListConfigs(secretKey string) ([]types.NamespacedName, error)
}
