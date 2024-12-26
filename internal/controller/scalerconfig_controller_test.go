package controller

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/quickube/QScaler/api/v1alpha1"
	"github.com/quickube/QScaler/internal/brokers"
	"github.com/quickube/QScaler/internal/mocks"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var _ = Describe("ScalerConfigReconciler", func() {
	var (
		scalerConfig     *v1alpha1.ScalerConfig
		secret           *corev1.Secret
		req              ctrl.Request
		BrokerMock2      *mocks.Broker
		namespace        = "default"
		scalerConfigName = "test-scalerconfig-2"
	)

	BeforeEach(func() {
		// Initialize test resources
		scalerConfig = &v1alpha1.ScalerConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      scalerConfigName,
				Namespace: namespace,
			},
			Spec: v1alpha1.ScalerConfigSpec{
				Type: fmt.Sprintf("%s/%s", namespace, scalerConfigName),
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

		req = ctrl.Request{NamespacedName: client.ObjectKey{Name: scalerConfig.Name, Namespace: scalerConfig.Namespace}}

		BrokerMock2 = &mocks.Broker{}
		brokers.BrokerRegistry[fmt.Sprintf("%s/%s", namespace, scalerConfigName)] = BrokerMock2
		BrokerMock2.On("IsConnected", mock.Anything).Return(true, nil)

		// Create resources in the fake Kubernetes cluster
		Expect(k8sClient.Create(context.Background(), scalerConfig)).To(Succeed())
		Expect(k8sClient.Create(context.Background(), secret)).To(Succeed())
		time.Sleep(5 * time.Second)
	})

	AfterEach(func() {
		By("Cleaning up resources")
		Expect(k8sManager.GetClient().Delete(ctx, scalerConfig)).To(Succeed())
		Expect(k8sManager.GetClient().Delete(ctx, secret)).To(Succeed())
	})

	Context("Reconcile", func() {
		It("should mark ScalerConfig as healthy if broker connects", func() {

			// Verify broker exists
			key := fmt.Sprintf("%s/%s", namespace, scalerConfigName)
			Expect(brokers.BrokerRegistry[key]).ToNot(BeNil())

			// Verify status update

			updated := &v1alpha1.ScalerConfig{}
			Expect(k8sClient.Get(context.Background(), req.NamespacedName, updated)).To(Succeed())
			Expect(updated.Status.Healthy).To(BeTrue())
		})
	})
})
