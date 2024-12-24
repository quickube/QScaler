/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/quickube/QScaler/api/v1alpha1"
	"github.com/quickube/QScaler/internal/brokers"
	"github.com/quickube/QScaler/internal/mocks"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Metrics Controller", func() {
	Context("Reconciliation logic", func() {
		var (
			resourceName         = "test-qworker"
			namespace            = "default"
			scalerConfigName     = "test-scalerconfig"
			qworkerResource      *v1alpha1.QWorker
			scalerConfigResource *v1alpha1.ScalerConfig
			BrokerMock           *mocks.Broker
		)

		BeforeEach(func() {
			By("Setting up a ScalerConfig resource")
			scalerConfigResource = &v1alpha1.ScalerConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      scalerConfigName,
					Namespace: namespace,
				},
				Spec: v1alpha1.ScalerConfigSpec{
					Type:   "test",
					Config: v1alpha1.ScalerTypeConfigs{},
				},
				Status: v1alpha1.ScalerConfigStatus{
					Healthy: true,
					Message: "ScalerConfig is healthy",
				},
			}

			By("Setting up a QWorker resource")
			qworkerResource = &v1alpha1.QWorker{
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
						MaxReplicas:     5,
						ScalingFactor:   1,
					},
				},
				Status: v1alpha1.QWorkerStatus{},
			}
			By("Creating resources")
			Expect(k8sClient.Create(ctx, scalerConfigResource)).To(Succeed())
			Expect(k8sClient.Create(ctx, qworkerResource)).To(Succeed())

			BrokerMock = &mocks.Broker{}
			configKey := fmt.Sprintf("%s/%s", namespace, scalerConfigName)
			brokers.BrokerRegistry[configKey] = BrokerMock
		})

		AfterEach(func() {
			By("Cleaning up resources")
			Expect(k8sClient.Delete(ctx, qworkerResource)).To(Succeed())
			Expect(k8sClient.Delete(ctx, scalerConfigResource)).To(Succeed())
		})

		It("should update QWorker status based on queue length", func() {
			By("Setting broker mocks")
			BrokerMock.On("GetQueueLength", mock.Anything, "test-queue").Return(3, nil)
			BrokerMock.On("IsConnected", mock.Anything).Return(true, nil)

			By("Simulating reconciliation")
			time.Sleep(5 * time.Second)

			By("Verifying QWorker status is updated")
			retrievedQWorker := &v1alpha1.QWorker{}
			expectKey := ctrlclient.ObjectKey{Namespace: namespace, Name: resourceName}
			Expect(k8sClient.Get(ctx, expectKey, retrievedQWorker)).To(Succeed())
			Expect(retrievedQWorker.Status.DesiredReplicas).To(Equal(3))
		})
	})
})
