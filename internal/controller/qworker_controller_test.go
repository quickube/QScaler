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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/quickube/QScaler/api/v1alpha1"
	"github.com/quickube/QScaler/internal/brokers"
	"github.com/quickube/QScaler/internal/mocks"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("QWorker Controller", func() {
	Context("Reconciliation logic", func() {
		It("should reconcile successfully and update QWorker status", func() {
			// Unique test identifiers
			testID := fmt.Sprintf("test-%d", time.Now().UnixNano())
			resourceName := fmt.Sprintf("qworker-%s", testID)
			scalerConfigName := fmt.Sprintf("scalerconfig-%s", testID)
			namespace := "default"
			configKey := fmt.Sprintf("%s/%s", namespace, scalerConfigName)

			// Create ScalerConfig resource
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
			Expect(k8sClient.Create(ctx, scalerConfigResource)).To(Succeed())

			// Create QWorker resource
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
						MaxReplicas:     5,
						ScalingFactor:   1,
					},
				},
				Status: v1alpha1.QWorkerStatus{},
			}
			Expect(k8sClient.Create(ctx, qworkerResource)).To(Succeed())

			// Mock Broker
			brokerMock := &mocks.Broker{}
			brokers.BrokerRegistry[configKey] = brokerMock
			brokerMock.On("GetQueueLength", mock.Anything, mock.Anything).Return(5, nil)
			brokerMock.On("IsConnected", mock.Anything).Return(true, nil)

			// Wait for reconciliation
			time.Sleep(5 * time.Second)

			// Check QWorker status
			retrievedQWorker := &v1alpha1.QWorker{}
			Expect(k8sClient.Get(ctx, ctrlclient.ObjectKey{Name: resourceName, Namespace: namespace}, retrievedQWorker)).To(Succeed())
			Expect(retrievedQWorker.Status.CurrentReplicas).To(BeNumerically(">=", 1))
			Expect(retrievedQWorker.Status.DesiredReplicas).To(BeNumerically("==", 5))

			// Cleanup resources
			Expect(k8sManager.GetClient().Delete(ctx, qworkerResource)).To(Succeed())
			Expect(k8sManager.GetClient().Delete(ctx, scalerConfigResource)).To(Succeed())
			delete(brokers.BrokerRegistry, configKey)
		})

		It("should scale up pods when needed", func() {
			// Unique test identifiers
			testID := fmt.Sprintf("test-%d", time.Now().UnixNano())
			resourceName := fmt.Sprintf("qworker-%s", testID)
			scalerConfigName := fmt.Sprintf("scalerconfig-%s", testID)
			namespace := "default"
			configKey := fmt.Sprintf("%s/%s", namespace, scalerConfigName)

			// Create ScalerConfig resource
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
			Expect(k8sClient.Create(ctx, scalerConfigResource)).To(Succeed())

			// Create QWorker resource
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
						MaxReplicas:     5,
						ScalingFactor:   1,
					},
				},
				Status: v1alpha1.QWorkerStatus{},
			}
			Expect(k8sClient.Create(ctx, qworkerResource)).To(Succeed())

			// Mock Broker
			brokerMock := &mocks.Broker{}
			brokers.BrokerRegistry[configKey] = brokerMock
			brokerMock.On("GetQueueLength", mock.Anything, mock.Anything).Return(5, nil)
			brokerMock.On("IsConnected", mock.Anything).Return(true, nil)

			// Wait for reconciliation
			time.Sleep(5 * time.Second)

			// Retrieve Pods
			podList := &corev1.PodList{}
			Expect(k8sClient.List(ctx, podList, ctrlclient.InNamespace(namespace))).To(Succeed())

			// Filter Pods by owner reference
			ownedPods := []corev1.Pod{}
			for _, pod := range podList.Items {
				for _, ownerRef := range pod.OwnerReferences {
					if ownerRef.Name == resourceName {
						ownedPods = append(ownedPods, pod)
					}
				}
			}

			// Delete all Pods
			for _, pod := range ownedPods {
				deleteOptions := &ctrlclient.DeleteOptions{
					GracePeriodSeconds: new(int64), // Pointer to 0
				}
				*deleteOptions.GracePeriodSeconds = 0

				Expect(k8sClient.Delete(ctx, &pod, deleteOptions)).To(Succeed())
			}

			time.Sleep(5 * time.Second)

			// Verify desired replicas match Pod count
			Expect(k8sClient.Get(ctx, ctrlclient.ObjectKeyFromObject(qworkerResource), qworkerResource)).To(Succeed())
			Expect(ownedPods).To(HaveLen(qworkerResource.Status.DesiredReplicas))

			// Cleanup resources
			Expect(k8sManager.GetClient().Delete(ctx, qworkerResource)).To(Succeed())
			Expect(k8sManager.GetClient().Delete(ctx, scalerConfigResource)).To(Succeed())
			delete(brokers.BrokerRegistry, configKey)
		})
	})
})
