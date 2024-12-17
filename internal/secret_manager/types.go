package secret_manager

import "github.com/quickube/QScaler/api/v1alpha1"

type SecretManager interface {
	Add(secret v1alpha1.ValueOrSecret) error
	Delete(secret v1alpha1.ValueOrSecret) error
	Get(secret v1alpha1.ValueOrSecret) (string, error)
	List() ([]v1alpha1.ValueOrSecret, error)
	Sync() error
}
