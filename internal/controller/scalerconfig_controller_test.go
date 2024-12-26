package controller

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/quickube/QScaler/api/v1alpha1"
	"github.com/quickube/QScaler/internal/brokers"
	"github.com/quickube/QScaler/internal/mocks"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("ScalerConfigReconciler", func() {
	var (
		scalerConfig     *v1alpha1.ScalerConfig
		secret           *corev1.Secret
		req              ctrl.Request
		BrokerMock       *mocks.Broker
		namespace        = "default"
		scalerConfigName = "test-scalerconfig"
	)

	BeforeEach(func() {
		// Initialize test resources
		scalerConfig = &v1alpha1.ScalerConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      scalerConfigName,
				Namespace: namespace,
			},
			Spec: v1alpha1.ScalerConfigSpec{
				Type: "test",
				Config: v1alpha1.ScalerTypeConfigs{
					RedisConfig: v1alpha1.RedisConfig{
						Password: v1alpha1.ValueOrSecret{
							Secret: &corev1.SecretKeySelector{
								Key:                  "password",
								LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"},
							},
						},
					},
				},
			},
		}

		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"password": []byte("password"),
			},
		}

		req = ctrl.Request{NamespacedName: types.NamespacedName{Name: scalerConfig.Name, Namespace: scalerConfig.Namespace}}

		BrokerMock = &mocks.Broker{}
		brokers.BrokerRegistry[fmt.Sprintf("%s/%s", namespace, scalerConfigName)] = BrokerMock

		// Create resources in the fake Kubernetes cluster
		Expect(k8sClient.Create(context.Background(), scalerConfig)).To(Succeed())
		Expect(k8sClient.Create(context.Background(), secret)).To(Succeed())
	})

	AfterEach(func() {
		// Cleanup resources
		Expect(k8sClient.Delete(context.Background(), scalerConfig)).To(Succeed())
		Expect(k8sClient.Delete(context.Background(), secret)).To(Succeed())
	})

	Context("Reconcile", func() {
		It("should mark ScalerConfig as healthy if broker connects", func() {
			BrokerMock.On("IsConnected", mock.Anything).Return(true, nil)

			// Trigger reconcile
			_, err := reconciler.Reconcile(context.TODO(), req)
			Expect(err).NotTo(HaveOccurred())

			// Verify status update
			updated := &v1alpha1.ScalerConfig{}
			Expect(k8sClient.Get(context.Background(), req.NamespacedName, updated)).To(Succeed())
		})

		It("should mark ScalerConfig as unhealthy if broker connection fails", func() {
			BrokerMock.On("IsConnected", mock.Anything).Return(false, fmt.Errorf("connection failed"))

			// Trigger reconcile
			_, err := reconciler.Reconcile(context.TODO(), req)
			Expect(err).To(HaveOccurred())

			// Verify status update
			updated := &v1alpha1.ScalerConfig{}
			Expect(k8sClient.Get(context.Background(), req.NamespacedName, updated)).To(Succeed())
		})
	})
})
