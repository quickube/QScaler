package metrics

import (
	"context"
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
		metricsServerInstance = &MetricsServer{
			client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			qworkers: &v1alpha1.QWorkerList{},
		}
	})
	return metricsServerInstance
}

func StartServer(mgr manager.Manager) {
	ctx := context.Background()
	_ = log.FromContext(ctx)

	server := getMetricsServer(mgr)
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
