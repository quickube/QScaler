package secret_manager

import (
	"github.com/quickube/QScaler/api/v1alpha1"
)

type SecretManager interface {
	Get(secret v1alpha1.ValueOrSecret) (string, error)
}
