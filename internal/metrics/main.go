package metrics

import (
	"context"
	metricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
	"sync"
	"time"

	"github.com/quickube/QScaler/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	metricsServerInstance *MetricsServer
	once                  sync.Once
)

func getMetricsServer(mgr manager.Manager) *MetricsServer {
	once.Do(func() {
		// Create a metrics client
		metricsClient, err := metricsv1beta1.NewForConfig(mgr.GetConfig())
		if err != nil {
			log.Log.Error(err, "error creating metrics client")
		}
		metricsServerInstance = &MetricsServer{
			client:        mgr.GetClient(),
			Scheme:        mgr.GetScheme(),
			metricsClient: metricsClient,
			qworkers:      &v1alpha1.QWorkerList{},
		}
	})
	return metricsServerInstance
}

func StartServer(ctx context.Context, mgr manager.Manager) {
	_ = log.FromContext(ctx)

	server := getMetricsServer(mgr)

	go run(ctx, *server)

}

func run(ctx context.Context, server MetricsServer) {

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
