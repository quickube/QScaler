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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/quickube/QScaler/api/v1alpha1"
	"github.com/quickube/QScaler/internal/brokers"
	"github.com/quickube/QScaler/internal/mocks"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("QWorker Controller", func() {
	Context("Reconciliation logic", func() {
		var (
			resourceName         = "test-qworker"
			namespace            = "default"
			scalerConfigName     = "test-scalerconfig"
			qworkerNamespaced    = types.NamespacedName{Name: resourceName, Namespace: namespace}
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
				Type: "test",
				Spec: map[string]string{
					"key": "value",
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

			By("Retrieve the QWorker resource to ensure it was created")
			createdQWorker := &v1alpha1.QWorker{}
			err := k8sClient.Get(ctx, ctrlclient.ObjectKey{
				Namespace: namespace,
				Name:      resourceName,
			}, createdQWorker)
			Expect(err).NotTo(HaveOccurred(), "Failed to fetch test QWorker resource")

			BrokerMock = &mocks.Broker{}
			brokers.BrokerRegistry["test"] = BrokerMock

		})

		AfterEach(func() {
			By("Cleaning up resources")
			Expect(k8sManager.GetClient().Delete(ctx, qworkerResource)).To(Succeed())
			Expect(k8sManager.GetClient().Delete(ctx, scalerConfigResource)).To(Succeed())
		})

		It("should reconcile successfully and update QWorker status", func() {
			By("Setting broker mocks")
			BrokerMock.On("GetQueueLength", mock.Anything, mock.Anything).Return(5, nil)
			BrokerMock.On("IsConnected", mock.Anything).Return(true, nil)

			time.Sleep(5 * time.Second)
			By("Checking QWorker status")
			retrievedQWorker := &v1alpha1.QWorker{}
			Expect(k8sClient.Get(ctx, qworkerNamespaced, retrievedQWorker)).To(Succeed())
			Expect(retrievedQWorker.Status.CurrentReplicas).To(BeNumerically(">=", 1))
			Expect(retrievedQWorker.Status.DesiredReplicas).To(BeNumerically("==", 5))

		})

		It("should scale up pods when needed", func() {
			By("Setting broker mocks")
			BrokerMock.On("GetQueueLength", mock.Anything, mock.Anything).Return(5, nil)
			BrokerMock.On("IsConnected", mock.Anything).Return(true, nil)
			time.Sleep(5 * time.Second)

			By("Retrieving all Pods in the namespace")
			podList := &corev1.PodList{}
			Expect(k8sClient.List(ctx, podList, ctrlclient.InNamespace(namespace))).To(Succeed())

			By("Filtering Pods by owner reference")
			ownedPods := []corev1.Pod{}
			for _, pod := range podList.Items {
				for _, ownerRef := range pod.OwnerReferences {
					if ownerRef.Name == resourceName {
						ownedPods = append(ownedPods, pod)
					}
				}
			}

			By("Deleting all Pods")
			for _, pod := range ownedPods {
				// Create delete options with GracePeriodSeconds set to 0
				deleteOptions := &ctrlclient.DeleteOptions{
					GracePeriodSeconds: new(int64), // A pointer to 0
				}
				*deleteOptions.GracePeriodSeconds = 0

				Expect(k8sClient.Delete(ctx, &pod, deleteOptions)).To(Succeed())
			}

			time.Sleep(5 * time.Second)

			By("Verifying the number of Pods matches the desired replicas")
			Expect(k8sClient.Get(ctx, ctrlclient.ObjectKeyFromObject(qworkerResource), qworkerResource)).To(Succeed())
			Expect(ownedPods).To(HaveLen(qworkerResource.Status.DesiredReplicas))
		})

	})
})
