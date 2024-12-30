package controller

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/mock"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/quickube/QScaler/api/v1alpha1"
	"github.com/quickube/QScaler/internal/brokers"
	"github.com/quickube/QScaler/internal/mocks"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("ScalerConfigReconciler", func() {
	Context("Reconcile", func() {
		It("should mark ScalerConfig as healthy if broker connects", func() {
			// Unique identifier for this test
			testID := fmt.Sprintf("test-%d", time.Now().UnixNano())
			scalerConfigName := fmt.Sprintf("scalerconfig-%s", testID)
			secretName := fmt.Sprintf("secret-%s", testID)
			brokerKey := fmt.Sprintf("%s/%s", "default", scalerConfigName)

			// Create test resources
			scalerConfig := &v1alpha1.ScalerConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      scalerConfigName,
					Namespace: "default",
				},
				Spec: v1alpha1.ScalerConfigSpec{
					Type: brokerKey,
					Config: v1alpha1.ScalerTypeConfigs{
						RedisConfig: v1alpha1.RedisConfig{
							Password: v1alpha1.ValueOrSecret{
								Secret: &corev1.SecretKeySelector{
									Key:                  "password",
									LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
								},
							},
						},
					},
				},
			}

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				Data: map[string][]byte{
					"password": []byte("password"),
				},
			}

			// Mock broker
			brokerMock := &mocks.Broker{}
			brokers.BrokerRegistry[brokerKey] = brokerMock
			brokerMock.On("IsConnected", mock.Anything).Return(true, nil)

			// Create resources in the fake Kubernetes cluster
			Expect(k8sClient.Create(context.Background(), scalerConfig)).To(Succeed())
			Expect(k8sClient.Create(context.Background(), secret)).To(Succeed())

			// Verify broker exists
			broker, err := brokers.GetBroker("default", scalerConfigName)
			Expect(broker).ToNot(BeNil(), "expected broker to be non-nil")
			Expect(err).To(BeNil(), "expected no error when retrieving broker")

			// Verify status update using Eventually
			updated := &v1alpha1.ScalerConfig{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKey{Name: scalerConfigName, Namespace: "default"}, updated)
				if err != nil {
					return false
				}
				return updated.Status.Healthy
			}, time.Second*10, time.Millisecond*500).Should(BeTrue(), "ScalerConfig should be marked as healthy")

			// Cleanup resources
			Expect(k8sManager.GetClient().Delete(ctx, scalerConfig)).To(Succeed())
			Expect(k8sManager.GetClient().Delete(ctx, secret)).To(Succeed())
			delete(brokers.BrokerRegistry, brokerKey)
		})

		It("should set healthy to false if the referenced secret is removed", func() {
			// Unique identifier for this test
			testID := fmt.Sprintf("test-%d", time.Now().UnixNano())
			scalerConfigName := fmt.Sprintf("scalerconfig-%s", testID)
			secretName := fmt.Sprintf("secret-%s", testID)
			brokerKey := fmt.Sprintf("%s/%s", "default", scalerConfigName)

			// Create test resources
			scalerConfig := &v1alpha1.ScalerConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      scalerConfigName,
					Namespace: "default",
				},
				Spec: v1alpha1.ScalerConfigSpec{
					Type: brokerKey,
					Config: v1alpha1.ScalerTypeConfigs{
						RedisConfig: v1alpha1.RedisConfig{
							Password: v1alpha1.ValueOrSecret{
								Secret: &corev1.SecretKeySelector{
									Key:                  "password",
									LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
								},
							},
						},
					},
				},
			}

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				Data: map[string][]byte{
					"password": []byte("password"),
				},
			}

			// Mock broker
			brokerMock := &mocks.Broker{}
			brokers.BrokerRegistry[brokerKey] = brokerMock
			brokerMock.On("IsConnected", mock.Anything).Return(false, nil)

			// Create resources in the fake Kubernetes cluster
			Expect(k8sClient.Create(context.Background(), scalerConfig)).To(Succeed())
			Expect(k8sClient.Create(context.Background(), secret)).To(Succeed())

			// Delete secret
			Expect(k8sClient.Delete(context.Background(), secret)).To(Succeed())

			// Verify status update using Eventually
			updated := &v1alpha1.ScalerConfig{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKey{Name: scalerConfigName, Namespace: "default"}, updated)
				if err != nil {
					return false
				}
				return updated.Status.Healthy
			}, time.Second*10, time.Millisecond*500).Should(BeFalse(), "ScalerConfig should be marked as unhealthy")

			// Cleanup resources
			Expect(k8sManager.GetClient().Delete(ctx, scalerConfig)).To(Succeed())
			delete(brokers.BrokerRegistry, brokerKey)
		})
	})
})
