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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type QWorkerSpec struct {
	PodSpec     corev1.PodSpec     `json:"podSpec"`
	ScaleConfig QWorkerScaleConfig `json:"scaleConfig,omitempty"`
}

type QWorkerStatus struct {
	CurrentReplicas    int    `json:"currentReplicas"`
	DesiredReplicas    int    `json:"desiredReplicas"`
	CurrentPodSpecHash string `json:"currentPodSpecHash"`
	// +kubebuilder:default={}
	MaxContainerResourcesUsage []corev1.ResourceList `json:"maxContainerResourcesUsage"`
}

type QWorkerScaleConfig struct {
	ScalerConfigRef string `json:"scalerConfigRef"`
	Queue           string `json:"queue"`
	MinReplicas     int    `json:"minReplicas"`
	MaxReplicas     int    `json:"maxReplicas"`
	ScalingFactor   int    `json:"scalingFactor"`
	// +kubebuilder:default=false
	ActivateVPA bool `json:"activateVPA"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type QWorker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   QWorkerSpec   `json:"spec,omitempty"`
	Status QWorkerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type QWorkerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []QWorker `json:"items"`
}

func init() {
	SchemeBuilder.Register(&QWorker{}, &QWorkerList{})
}
