package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/quickube/QScaler/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	metricsServerInstance *MetricsServer
	once                  sync.Once
)

func getMetricsServer(client client.Client) *MetricsServer {
	once.Do(func() {
		metricsServerInstance = &MetricsServer{
			client:   client,
			qworkers: &v1alpha1.QWorkerList{},
		}
	})
	return metricsServerInstance
}

func StartServer() {
	ctx := context.Background()
	_ = log.FromContext(ctx)

	kubeClient, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		log.Log.Error(err, "failed to create Kubernetes client")
		return
	}

	server := getMetricsServer(kubeClient)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Log.Info("shutting down MetricsServer")
			return
		case <-ticker.C:
			log.Log.Info("running MetricsServer reconciliation")
			if err := server.Run(ctx); err != nil {
				log.Log.Error(err, "failed to run MetricsServer reconciliation")
			}
		}
	}
}
