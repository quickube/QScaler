package metrics

import (
	"context"
	"fmt"
	"github.com/quickube/QScaler/api/v1alpha1"
	"github.com/quickube/QScaler/internal/brokers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsv1beta1client "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	thresholdPercent = 10.0
)

type MetricsServer struct {
	client        client.Client
	qworkers      *v1alpha1.QWorkerList
	Scheme        *runtime.Scheme
	metricsClient *metricsv1beta1client.MetricsV1beta1Client
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

		if qworker.Spec.ScaleConfig.ActivateVPA {
			err = s.RightSizeContainers(ctx, &qworker)
			if err != nil {
				return err
			}
		}

		if err = s.client.Status().Update(ctx, &qworker); err != nil {
			log.Log.Error(err, "Failed to update QWorker status")
		}
	}
	return nil
}

func (s *MetricsServer) RightSizeContainers(ctx context.Context, qworker *v1alpha1.QWorker) error {
	var podList corev1.PodList
	var err error

	if err = s.client.List(ctx, &podList, client.InNamespace(qworker.Namespace), client.MatchingFields{"metadata.ownerReferences.name": qworker.Name}); err != nil {
		return err
	}

	if len(podList.Items) == 0 {
		log.Log.Info(fmt.Sprintf("No pods found in namespace: %s", qworker.Namespace))
		return nil
	}

	numberOfContainersPerPod := len(podList.Items[0].Spec.Containers)
	for container := 0; container < numberOfContainersPerPod; container++ {
		if len(qworker.Status.MaxContainerResourcesUsage) <= 0 {
			qworker.Status.MaxContainerResourcesUsage =
				append(qworker.Status.MaxContainerResourcesUsage, corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("0"),
					corev1.ResourceMemory: resource.MustParse("0"),
				})
		}
		containerMaxCpuUsage := qworker.Status.MaxContainerResourcesUsage[container]["cpu"]
		containerMaxMemoryUsage := qworker.Status.MaxContainerResourcesUsage[container]["memory"]

		for _, pod := range podList.Items {
			var podMetrics *metricsv1beta1.PodMetrics
			podMetrics, err = s.metricsClient.PodMetricses(pod.Namespace).Get(ctx, pod.Name, v1.GetOptions{})
			if err != nil {
				return err
			}
			for i, _ := range podMetrics.Containers {
				// CPU usage
				cpu := podMetrics.Containers[i].Usage.Cpu()
				if exceedsThreshold(containerMaxCpuUsage, *cpu, thresholdPercent) {
					log.Log.Info(fmt.Sprintf("changing qworker %s CPU to %s from %s",
						cpu.String(), qworker.Name, containerMaxCpuUsage.String()))
					containerMaxCpuUsage = *cpu
				}

				// Memory usage
				memory := podMetrics.Containers[i].Usage.Memory()
				if exceedsThreshold(containerMaxMemoryUsage, *memory, thresholdPercent) {
					log.Log.Info(fmt.Sprintf("changing qworker %s Memory to %s from %s",
						memory.String(), qworker.Name, containerMaxMemoryUsage.String()))
					containerMaxMemoryUsage = *memory
				}
			}
		}
		qworker.Status.MaxContainerResourcesUsage[container]["cpu"] = containerMaxCpuUsage
		qworker.Status.MaxContainerResourcesUsage[container]["memory"] = containerMaxMemoryUsage
	}
	return nil
}

func exceedsThreshold(current, new resource.Quantity, thresholdPercent float64) bool {
	currentValue := float64(current.MilliValue())
	newValue := float64(new.MilliValue())

	// If the current value is 0, any non-zero new value exceeds the threshold
	if currentValue == 0 {
		return newValue > 0
	}

	// Calculate the percentage increase
	percentageIncrease := ((newValue - currentValue) / currentValue) * 100
	return percentageIncrease > thresholdPercent
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
