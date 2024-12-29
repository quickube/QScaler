package secret_manager

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/quickube/QScaler/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	// Singleton instance and sync.Once
	secretManagerInstance SecretManager
	once                  sync.Once
)

type SecretManagerInst struct {
	client    kubernetes.Interface
	namespace string
}

func NewClient() (SecretManager, error) {
	var initErr error

	once.Do(func() {
		// Initialize configuration
		config, err := rest.InClusterConfig()
		if err != nil {
			initErr = fmt.Errorf("error getting client config: %v", err)
			return
		}

		// Create Kubernetes clientset
		clientset := kubernetes.NewForConfigOrDie(config)

		// Read namespace from the service account
		namespacePath := "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
		namespace, err := os.ReadFile(namespacePath)
		if err != nil {
			initErr = fmt.Errorf("could not read namespace file %s: %w", namespacePath, err)
			return
		}

		// Create the singleton instance
		secretManager := &SecretManagerInst{
			client:    clientset,
			namespace: string(namespace),
		}

		secretManagerInstance = secretManager
	})

	return secretManagerInstance, initErr
}

func (s *SecretManagerInst) Get(secret v1alpha1.ValueOrSecret) (string, error) {

	if secret.Secret != nil {
		var ok bool
		var bytes []byte
		k8sSecret, err := s.client.CoreV1().Secrets(s.namespace).Get(context.Background(), secret.Secret.Name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}

		if bytes, ok = k8sSecret.Data[secret.Secret.Key]; !ok {
			return "", fmt.Errorf("missing key %s in secret %s", secret.Secret.Key, s.namespace)
		}

		return string(bytes), nil
	}

	return secret.Value, nil
}
