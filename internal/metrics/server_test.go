package metrics

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/quickube/QScaler/api/v1alpha1"
	"github.com/quickube/QScaler/internal/brokers"
	"github.com/quickube/QScaler/internal/mocks"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
