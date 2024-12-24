package secret_informer

import (
	"context"
	"fmt"
	"github.com/quickube/QScaler/api/v1alpha1"
	"github.com/quickube/QScaler/internal/brokers"
	"github.com/quickube/QScaler/internal/secret_manager"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

type SecretInformer struct {
	client        *kubernetes.Clientset
	brokerClient  client.Client
	namespace     string
	secretManager secret_manager.SecretManager
}

func StartSecretInformer(client client.Client) error {
	// Initialize configuration
	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("error getting client config: %v", err)
	}

	// Create Kubernetes clientset
	clientset := kubernetes.NewForConfigOrDie(config)

	// Read namespace from the service account
	namespacePath := "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	namespace, err := os.ReadFile(namespacePath)
	if err != nil {
		return fmt.Errorf("could not read namespace file %s: %w", namespacePath, err)
	}

	secretManager, err := secret_manager.NewClient()
	if err != nil {
		return fmt.Errorf("could not create secret manager: %w", err)
	}

	secretInformer := &SecretInformer{
		client:        clientset,
		brokerClient:  client,
		namespace:     string(namespace),
		secretManager: secretManager,
	}

	go secretInformer.start()

	return nil

}
func (s *SecretInformer) start() {

	// Create a shared informer factory for the given namespace
	factory := informers.NewSharedInformerFactoryWithOptions(s.client, 2*time.Minute, informers.WithNamespace(s.namespace))

	// Create the Secret informer
	secretInformer := factory.Core().V1().Secrets()

	//secretLister := secretInformer.Lister()

	// Add event handlers for the informer
	secretInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    s.handleAdd,
		UpdateFunc: s.handleUpdate,
		DeleteFunc: s.handleDelete,
	})

	// Start the informer and wait for the cache to sync
	stopCh := make(chan struct{})
	defer close(stopCh)
	factory.Start(stopCh)

	if !cache.WaitForCacheSync(stopCh, secretInformer.Informer().HasSynced) {
		log.Log.Info(fmt.Sprintf("waiting for cache to sync"))
		return
	}

	log.Log.Info(fmt.Sprintf("Secret informer started and synced"))
}

func (s *SecretInformer) handleAdd(obj interface{}) {
	secret := obj.(*corev1.Secret)
	//err := s.sync(secret)
	//if err != nil {
	//	log.Log.Error(fmt.Errorf("error syncing secret"), err.Error())
	//	return
	//}
	//log.Log.Info("Secret Added: ", secret.Name)
	fmt.Sprintf("Secret Added: %s", secret.Name)
}

func (s *SecretInformer) handleUpdate(oldObj, newObj interface{}) {
	secret := oldObj.(*corev1.Secret)
	err := s.sync(secret)
	if err != nil {
		log.Log.Error(fmt.Errorf("error syncing secret"), err.Error())
		return
	}
	log.Log.Info("Secret Updated: ", secret.Name)
}

func (s *SecretInformer) handleDelete(obj interface{}) {
	secret := obj.(*corev1.Secret)
	err := s.sync(secret)
	if err != nil {
		log.Log.Error(fmt.Errorf("error syncing secret"), err.Error())
		return
	}
	log.Log.Info("Secret Deleted: ", secret.Name)
}

func (s *SecretInformer) sync(secret *corev1.Secret) error {
	configList, err := s.secretManager.ListConfigs(secret.Name)
	if err != nil {
		return err
	}
	if len(configList) == 0 {
		return nil
	}
	for _, config := range configList {
		err = s.secretManager.Sync(config)
		if err != nil {
			return err
		}

		scalerConfig := &v1alpha1.ScalerConfig{}
		if err = s.brokerClient.Get(context.Background(), config, scalerConfig); err != nil {
			return err
		}

		_, err = brokers.UpdateBroker(scalerConfig)
		if err != nil {
			return err
		}
	}

	return nil
}
