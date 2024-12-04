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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type ScalerConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Type              string             `json:"type"`
	Spec              ScalerSpec         `json:"spec"`
	Status            ScalerConfigStatus `json:"status,omitempty"`
}

type ScalerSpec struct {
	Host             string `json:"host"`
	Port             int32  `json:"port"`
	Password         string `json:"password"`
	IsPasswordSecret bool   `json:"isPasswordSecret"`
}

type ScalerConfigStatus struct {
	Healthy bool   `json:"healthy,omitempty"`
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true

type ScalerConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ScalerConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ScalerConfig{}, &ScalerConfigList{})
}
