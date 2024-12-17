package secret_manager

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"sync"

	"github.com/quickube/QScaler/api/v1alpha1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	// Singleton instance and sync.Once
	secretManagerInstance SecretManager
	once                  sync.Once
)

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
		secretManagerInstance = &SecretManagerInst{
			registry:  make(map[v1alpha1.ValueOrSecret]string),
			client:    clientset,
			namespace: string(namespace),
			mutex:     sync.Mutex{},
		}
	})

	return secretManagerInstance, initErr
}

type SecretManagerInst struct {
	registry  map[v1alpha1.ValueOrSecret]string
	client    kubernetes.Interface
	namespace string
	mutex     sync.Mutex
}

func (s *SecretManagerInst) Add(secret v1alpha1.ValueOrSecret) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var value []byte
	var ok bool

	if _, ok = s.registry[secret]; !ok {
		return nil
	}
	k8sSecret, err := s.client.CoreV1().Secrets(s.namespace).Get(context.Background(), secret.ValueFrom.SecretKeyRef.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if value, ok = k8sSecret.Data[secret.ValueFrom.SecretKeyRef.Key]; !ok {
		return fmt.Errorf("missing key %s in secret %s", secret.ValueFrom.SecretKeyRef.Key, s.namespace)
	}

	s.registry[secret] = string(value)

	return nil
}

func (s *SecretManagerInst) Delete(secret v1alpha1.ValueOrSecret) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, ok := s.registry[secret]; !ok {
		return nil
	}

	delete(s.registry, secret)

	return nil
}

func (s *SecretManagerInst) Get(secret v1alpha1.ValueOrSecret) (string, error) {
	return s.registry[secret], nil
}

func (s *SecretManagerInst) List() ([]v1alpha1.ValueOrSecret, error) {
	var result []v1alpha1.ValueOrSecret

	for k, _ := range s.registry {
		result = append(result, k)
	}
	return result, nil
}

func (s *SecretManagerInst) Sync() error {
	for secret, _ := range s.registry {
		k8sSecret, err := s.client.CoreV1().Secrets(s.namespace).Get(context.Background(), secret.ValueFrom.SecretKeyRef.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		value, ok := k8sSecret.Data[secret.ValueFrom.SecretKeyRef.Key]
		if !ok {
			return fmt.Errorf("missing key %s in secret %s", secret.ValueFrom.SecretKeyRef.Key, s.namespace)
		}
		s.registry[secret] = string(value)
	}

	return nil
}
