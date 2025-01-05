package metrics

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/quickube/QScaler/api/v1alpha1"
	"github.com/quickube/QScaler/internal/brokers"
	"github.com/quickube/QScaler/internal/mocks"
	assertion "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	fake2 "k8s.io/metrics/pkg/client/clientset/versioned/fake"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestMetricsServer_Sync(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	// create qworker resource
	qworker := &v1alpha1.QWorker{
		Spec: v1alpha1.QWorkerSpec{
			ScaleConfig: v1alpha1.QWorkerScaleConfig{
				ScalerConfigRef: "test-scaler-config",
				Queue:           "test-queue",
			},
		},
	}

	// create qworkerlist
	qworkerList := &v1alpha1.QWorkerList{
		Items: []v1alpha1.QWorker{*qworker},
	}

	// create fake client
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithLists(qworkerList).
		Build()

	// create metrics server
	server := MetricsServer{
		client: client,
		Scheme: scheme,
	}

	ctx := context.Background()
	err := server.Sync(ctx)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// verify that the sync add 1 qworker to the qworkers list
	if len(server.qworkers.Items) != 1 {
		t.Errorf("Expected 1 QWorker, got %d", len(server.qworkers.Items))
	}
}

func TestMetricsServer_Run(t *testing.T) {
	// test params
	testID := fmt.Sprintf("test-%d", time.Now().UnixNano())
	resourceName := fmt.Sprintf("qworker-%s", testID)
	scalerConfigName := fmt.Sprintf("scalerconfig-%s", testID)
	namespace := "default"
	configKey := fmt.Sprintf("%s/%s", namespace, scalerConfigName)

	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	// create qworker resource
	qworkerResource := &v1alpha1.QWorker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: namespace,
		},
		Spec: v1alpha1.QWorkerSpec{
			PodSpec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "worker-container",
						Image: "busybox",
					},
				},
			},
			ScaleConfig: v1alpha1.QWorkerScaleConfig{
				ScalerConfigRef: scalerConfigName,
				Queue:           "test-queue",
				MinReplicas:     1,
				MaxReplicas:     10,
				ScalingFactor:   1,
			},
		},
		Status: v1alpha1.QWorkerStatus{},
	}

	// create scalerconfig resource
	scalerConfigResource := &v1alpha1.ScalerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scalerConfigName,
			Namespace: namespace,
		},
		Spec: v1alpha1.ScalerConfigSpec{
			Type:   configKey,
			Config: v1alpha1.ScalerTypeConfigs{},
		},
	}

	// create empty qworkerlist
	qworkerList := &v1alpha1.QWorkerList{
		Items: []v1alpha1.QWorker{},
	}

	// create fake client with the resources
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(qworkerResource, scalerConfigResource).
		WithStatusSubresource(qworkerResource).
		Build()

	server := MetricsServer{
		client:   client,
		Scheme:   scheme,
		qworkers: qworkerList,
	}

	brokerMock := &mocks.Broker{}
	brokers.BrokerRegistry[configKey] = brokerMock
	brokerMock.On("GetQueueLength", mock.Anything, mock.Anything).Return(10, nil)

	// Run the server
	ctx := context.Background()
	err := server.Run(ctx)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify updates
	updatedQWorker := &v1alpha1.QWorker{}
	err = client.Get(ctx, ctrlclient.ObjectKeyFromObject(qworkerResource), updatedQWorker)
	if err != nil {
		t.Fatalf("Failed to get updated QWorker: %v", err)
	}

	if updatedQWorker.Status.DesiredReplicas != 10 {
		t.Errorf("Expected desired replicas to be 10, got %d", updatedQWorker.Status.DesiredReplicas)
	}
}

func TestExceedsThreshold(t *testing.T) {
	tests := []struct {
		name             string
		current          string
		new              string
		thresholdPercent float64
		expected         bool
	}{
		{
			name:             "New value exceeds threshold",
			current:          "100m",
			new:              "150m",
			thresholdPercent: 40.0,
			expected:         true,
		},
		{
			name:             "New value does not exceed threshold",
			current:          "100m",
			new:              "120m",
			thresholdPercent: 40.0,
			expected:         false,
		},
		{
			name:             "Current value is zero and new value is non-zero",
			current:          "0",
			new:              "100m",
			thresholdPercent: 50.0,
			expected:         true,
		},
		{
			name:             "Current value and new value are both zero",
			current:          "0",
			new:              "0",
			thresholdPercent: 50.0,
			expected:         false,
		},
		{
			name:             "New value is less than current value",
			current:          "200m",
			new:              "100m",
			thresholdPercent: 50.0,
			expected:         false,
		},
		{
			name:             "Threshold is zero, any increase should exceed",
			current:          "100m",
			new:              "110m",
			thresholdPercent: 0.0,
			expected:         true,
		},
		{
			name:             "No increase in values",
			current:          "100m",
			new:              "100m",
			thresholdPercent: 10.0,
			expected:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentQuantity := resource.MustParse(tt.current)
			newQuantity := resource.MustParse(tt.new)

			result := exceedsThreshold(currentQuantity, newQuantity, tt.thresholdPercent)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRightSizeContainers(t *testing.T) {
	assert := assertion.New(t)
	tests := []struct {
		name            string
		qworker         *v1alpha1.QWorker
		expectedQworker *v1alpha1.QWorker
		podList         []ctrlclient.Object
		metricsData     map[string]*metricsv1beta1.PodMetrics
		expectedError   bool
	}{
		{
			name: "No pods found",
			qworker: &v1alpha1.QWorker{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-qworker",
				},
			},
			expectedQworker: &v1alpha1.QWorker{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-qworker",
				},
			},
			podList:       []ctrlclient.Object{},
			metricsData:   map[string]*metricsv1beta1.PodMetrics{},
			expectedError: false,
		},
		{
			name: "Pod metrics exceed threshold",
			qworker: &v1alpha1.QWorker{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-qworker",
				},
				Status: v1alpha1.QWorkerStatus{
					MaxContainerResourcesUsage: []corev1.ResourceList{
						{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("512Mi"),
						},
					},
				},
			},
			expectedQworker: &v1alpha1.QWorker{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-qworker",
				},
				Status: v1alpha1.QWorkerStatus{
					MaxContainerResourcesUsage: []corev1.ResourceList{
						{
							corev1.ResourceCPU:    resource.MustParse("600m"),
							corev1.ResourceMemory: resource.MustParse("1Gi"),
						},
					},
				},
			},
			podList: []ctrlclient.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-namespace",
						Name:      "test-pod",
						OwnerReferences: []metav1.OwnerReference{
							{Name: "test-qworker"},
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "test-container"}},
					},
				},
			},
			metricsData: map[string]*metricsv1beta1.PodMetrics{
				"test-namespace_test-pod": {
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-namespace",
						Name:      "test-pod",
					},
					Containers: []metricsv1beta1.ContainerMetrics{
						{
							Name: "test-container",
							Usage: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("600m"),
								corev1.ResourceMemory: resource.MustParse("1Gi"),
							},
						},
					},
				},
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			_ = metricsv1beta1.AddToScheme(scheme)

			// Create fake client for pods
			clientBuilder := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.podList...)
			clientBuilder.WithIndex(&corev1.Pod{}, "metadata.ownerReferences.name", func(obj ctrlclient.Object) []string {
				pod, ok := obj.(*corev1.Pod)
				if !ok {
					return []string{}
				}
				var results []string
				for _, ownerRef := range pod.OwnerReferences {
					results = append(results, ownerRef.Name)
				}
				return results
			})
			client := clientBuilder.Build()

			// Create fake metrics client with reactors
			metricsFakeClient := fake2.Clientset{}
			metricsClient := metricsFakeClient.MetricsV1beta1()
			metricsFakeClient.PrependReactor("get", "podmetrics", func(action k8stesting.Action) (bool, runtime.Object, error) {
				getAction := action.(k8stesting.GetAction)
				if key, exists := tt.metricsData[getAction.GetNamespace()+"_"+getAction.GetName()]; exists {
					return true, key, nil
				}
				return true, nil, fmt.Errorf("Pod metrics not found for %s/%s", getAction.GetNamespace(), getAction.GetName())
			})

			s := MetricsServer{
				client:        client,
				metricsClient: metricsClient,
			}

			err := s.RightSizeContainers(ctx, tt.qworker)
			assert.Equal(tt.expectedError, err != nil)

			if len(tt.metricsData) > 0 {
				// assert if not equal
				assert.False(tt.qworker.Status.MaxContainerResourcesUsage[0].Cpu().
					Equal(*tt.expectedQworker.Status.MaxContainerResourcesUsage[0].Cpu()))
				assert.False(tt.qworker.Status.MaxContainerResourcesUsage[0].Memory().
					Equal(*tt.expectedQworker.Status.MaxContainerResourcesUsage[0].Memory()))
			}

		})
	}
}
