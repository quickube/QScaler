package metrics

import (
	"context"
	"fmt"

	"github.com/quickube/QScaler/api/v1alpha1"
	"github.com/quickube/QScaler/internal/brokers"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type MetricsServer struct {
	client   client.Client
	qworkers *v1alpha1.QWorkerList
	Scheme   *runtime.Scheme
}

func (s *MetricsServer) Run(ctx context.Context) error {
	log.Log.Info("Starting QScaler Metrics Server")

	var BrokerClient brokers.Broker
	var QueueLength int
	var err error

	err = s.Sync(ctx)
	if err != nil {
		return err
	}

	if len(s.qworkers.Items) == 0 {
		log.Log.Info("No qworkers found!")
		return nil
	}
	for _, qworker := range s.qworkers.Items {
		var scalerConfig v1alpha1.ScalerConfig
		namespacedName := client.ObjectKey{Name: qworker.Spec.ScaleConfig.ScalerConfigRef, Namespace: qworker.ObjectMeta.Namespace}
		if err = s.client.Get(ctx, namespacedName, &scalerConfig); err != nil {
			log.Log.Error(err, "Failed to get ScalerConfig", "namespacedName", namespacedName.String())
		}

		BrokerClient, err = brokers.NewBroker(&scalerConfig)
		if err != nil {
			log.Log.Error(err, "Failed to create broker client")
		}

		QueueLength, err = BrokerClient.GetQueueLength(&ctx, qworker.Spec.ScaleConfig.Queue)
		if err != nil {
			log.Log.Error(err, "Failed to get queue length")
		}
		log.Log.Info(fmt.Sprintf("current queue length: %d", QueueLength))

		desiredPodsAmount := min(
			max(QueueLength*qworker.Spec.ScaleConfig.ScalingFactor, qworker.Spec.ScaleConfig.MinReplicas),
			qworker.Spec.ScaleConfig.MaxReplicas)
		log.Log.Info(fmt.Sprintf("desired amount: %d", desiredPodsAmount))
		qworker.Status.DesiredReplicas = desiredPodsAmount

		if err = s.client.Status().Update(ctx, &qworker); err != nil {
			log.Log.Error(err, "Failed to update QWorker status")
		}
	}
	return nil
}

func (s *MetricsServer) Sync(ctx context.Context) error {
	_ = log.FromContext(ctx)
	qworkerList := &v1alpha1.QWorkerList{}
	err := s.client.List(ctx, qworkerList, &client.ListOptions{})
	if err != nil {
		log.Log.Error(err, "unable to list QWorker resources")
		return err
	}
	s.qworkers = qworkerList
	log.Log.Info("successfully synchronized QWorkers", "count", len(qworkerList.Items))
	return nil
}
