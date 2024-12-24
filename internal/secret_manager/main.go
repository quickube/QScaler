package secret_manager

import (
	"context"
	"fmt"
	"github.com/quickube/QScaler/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"slices"
	"sync"
)

var (
	// Singleton instance and sync.Once
	secretManagerInstance SecretManager
	once                  sync.Once
)

type SecretManagerInst struct {
	registry  map[types.NamespacedName]map[v1alpha1.ValueOrSecret]string
	client    kubernetes.Interface
	namespace string
	mutex     sync.Mutex
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
			registry:  make(map[types.NamespacedName]map[v1alpha1.ValueOrSecret]string),
			client:    clientset,
			namespace: string(namespace),
			mutex:     sync.Mutex{},
		}

		secretManagerInstance = secretManager
	})

	return secretManagerInstance, initErr
}

func (s *SecretManagerInst) Add(configName types.NamespacedName, secret v1alpha1.ValueOrSecret) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var value []byte
	var ok bool

	if _, ok = s.registry[configName][secret]; !ok {
		return nil
	}
	k8sSecret, err := s.client.CoreV1().Secrets(s.namespace).Get(context.Background(), secret.ValueFrom.SecretKeyRef.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if value, ok = k8sSecret.Data[secret.ValueFrom.SecretKeyRef.Key]; !ok {
		return fmt.Errorf("missing key %s in secret %s", secret.ValueFrom.SecretKeyRef.Key, s.namespace)
	}

	s.registry[configName][secret] = string(value)

	return nil
}

func (s *SecretManagerInst) Delete(configName types.NamespacedName, secret v1alpha1.ValueOrSecret) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.registry[configName], secret)

	return nil
}

func (s *SecretManagerInst) Get(configName types.NamespacedName, secret v1alpha1.ValueOrSecret) (string, error) {

	value, ok := s.registry[configName][secret]

	if !ok {
		return "", fmt.Errorf("secret %s not found in namespace %s", secret, s.namespace)
	}
	return value, nil
}

func (s *SecretManagerInst) Sync(configName types.NamespacedName) error {

	var ok bool
	var value []byte

	if _, ok = s.registry[configName]; !ok {
		for secret, _ := range s.registry[configName] {
			if secret.ValueFrom.SecretKeyRef != nil {
				k8sSecret, err := s.client.CoreV1().Secrets(s.namespace).Get(context.Background(), secret.ValueFrom.SecretKeyRef.Name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				value, ok = k8sSecret.Data[secret.ValueFrom.SecretKeyRef.Key]
				if !ok {
					return fmt.Errorf("missing key %s in secret %s", secret.ValueFrom.SecretKeyRef.Key, s.namespace)
				}
				s.registry[configName][secret] = string(value)
			}
		}

		return nil
	}
	return nil
}

func (s *SecretManagerInst) ListConfigs(secretKey string) ([]types.NamespacedName, error) {
	var configList []types.NamespacedName
	for config := range s.registry {
		for secret, _ := range s.registry[config] {
			if secret.ValueFrom.SecretKeyRef != nil && secret.ValueFrom.SecretKeyRef.Name == secretKey {
				if !slices.Contains(configList, config) {
					configList = append(configList, config)
				}
			}
		}
	}
	if len(configList) == 0 {
		return nil, fmt.Errorf("secret %s not found in namespace %s", secretKey, s.namespace)
	}

	return configList, nil
}
