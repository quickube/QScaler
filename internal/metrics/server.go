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
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsv1beta1client "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
	"time"
)

var (
	thresholdPercent = 10.0
)

type MetricsServer struct {
	client        client.Client
	qworkers      *v1alpha1.QWorkerList
	Scheme        *runtime.Scheme
	metricsClient metricsv1beta1client.MetricsV1beta1Interface
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
		if len(qworker.Status.MaxContainerResourcesUsage) == 0 {
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
			for i := range podMetrics.Containers {
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

func (s *MetricsServer) startOOMKillEventInformer(ctx context.Context) {
	log.Log.Info("Starting OOMKill Event Informer")
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Log.Error(err, "Failed to load in-cluster config")
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Log.Error(err, "Failed to create Kubernetes clientset")
	}

	// Create a shared informer factory
	informerFactory := informers.NewSharedInformerFactory(kubeClient, 30*time.Second)
	eventInformer := informerFactory.Core().V1().Events().Informer()

	// Add event handler
	_, err = eventInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			event := obj.(*corev1.Event)
			if event.Reason == "Killing" && strings.Contains(event.Message, "due to OOM") {
				log.Log.Info(fmt.Sprintf("OOMKill event detected: %s/%s - %s", event.Namespace, event.InvolvedObject.Name, event.Message))
				s.handleOOMKillEvent(ctx, event)
			}
		},
	})
	if err != nil {
		fmt.Println(fmt.Sprintf(err.Error()))
	}

	// Start the informer
	stopCh := make(chan struct{})
	go func() {
		<-ctx.Done()
		close(stopCh)
	}()
	defer close(stopCh)

	informerFactory.Start(stopCh)
	informerFactory.WaitForCacheSync(stopCh)
	log.Log.Info("OOMKill Event Informer started")
}

func (s *MetricsServer) handleOOMKillEvent(ctx context.Context, event *corev1.Event) {
	log.Log.Info(fmt.Sprintf("Handling OOMKill event for Pod: %s in Namespace: %s", event.InvolvedObject.Name, event.Namespace))

	// Retrieve the affected pod
	var pod corev1.Pod
	err := s.client.Get(ctx, client.ObjectKey{
		Namespace: event.Namespace,
		Name:      event.InvolvedObject.Name,
	}, &pod)
	if err != nil {
		log.Log.Error(err, "Failed to get Pod", "PodName", event.InvolvedObject.Name, "Namespace", event.Namespace)
		return
	}

	// Check if the pod belongs to a QWorker
	for _, ownerRef := range pod.OwnerReferences {
		if ownerRef.Kind == "QWorker" {
			// Fetch the QWorker
			var qworker v1alpha1.QWorker
			err = s.client.Get(ctx, client.ObjectKey{
				Namespace: pod.Namespace,
				Name:      ownerRef.Name,
			}, &qworker)
			if err != nil {
				log.Log.Error(err, "Failed to get QWorker", "QWorkerName", ownerRef.Name, "Namespace", pod.Namespace)
				return
			}

			// Increase the MaxMemory by 10%
			err = s.increaseMaxMemory(ctx, &qworker)
			if err != nil {
				log.Log.Error(err, "Failed to increase MaxMemory for QWorker", "QWorkerName", qworker.Name)
			}
			return
		}
	}
}

func (s *MetricsServer) increaseMaxMemory(ctx context.Context, qworker *v1alpha1.QWorker) error {
	// Iterate over the container resource usages and increase memory by 10%
	for i := range qworker.Status.MaxContainerResourcesUsage {
		currentMemory := qworker.Status.MaxContainerResourcesUsage[i][corev1.ResourceMemory]
		tenPercent := currentMemory.Value() / 10
		newMemory := currentMemory.DeepCopy()
		newMemory.Add(*resource.NewQuantity(tenPercent, resource.BinarySI))

		qworker.Status.MaxContainerResourcesUsage[i][corev1.ResourceMemory] = newMemory

		log.Log.Info(fmt.Sprintf("Increased max memory for QWorker %s by 10%% to %s",
			qworker.Name, newMemory.String()))
	}

	// Update the QWorker's status
	return s.client.Status().Update(ctx, qworker)
}
