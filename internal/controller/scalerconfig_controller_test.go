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
	"context"
	"github.com/quickube/QScaler/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestFetchSecretsFromReferences(t *testing.T) {
	globalConfig := &v1alpha1.ScalerConfig{
		Spec: v1alpha1.ScalerConfigSpec{
			Type:   "test-type",
			Config: v1alpha1.ScalerTypeConfigs{}}}

	t.Run("Fail on Secret not found", func(t *testing.T) {
		// Prepare
		mockClient := runtimeclient.NewFakeClient()
		controllerReconciler := ScalerConfigReconciler{
			Client: mockClient,
			Scheme: mockClient.Scheme(),
		}
		ctx := context.Background()

		config := globalConfig.DeepCopy()
		config.Spec.Config = v1alpha1.ScalerTypeConfigs{
			v1alpha1.RedisConfig{
				Host: "host",
				Port: "0",
				Password: v1alpha1.ValueOrSecret{
					Value: "",
					ValueFrom: v1alpha1.ValueSources{
						SecretKeyRef: corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "missing-secret"},
							Key:                  "test-key",
						},
					},
				},
			},
		}
		// Execute
		err := controllerReconciler.fetchSecretsFromReferences(ctx, config)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, err.Error(), "secrets \"missing-secret\" not found")
	})
	t.Run("Fail on key not in secret", func(t *testing.T) {
		// Prepare
		fakeSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "some-secret"}, Data: map[string][]byte{"other-key": {}}}
		mockClient := runtimeclient.NewClientBuilder().WithObjects(fakeSecret).Build()
		controllerReconciler := ScalerConfigReconciler{
			Client: mockClient,
			Scheme: mockClient.Scheme(),
		}
		ctx := context.Background()

		config := globalConfig.DeepCopy()
		config.Spec.Config = v1alpha1.ScalerTypeConfigs{
			v1alpha1.RedisConfig{
				Host: "host",
				Port: "0",
				Password: v1alpha1.ValueOrSecret{
					Value: "",
					ValueFrom: v1alpha1.ValueSources{
						SecretKeyRef: corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "some-secret"},
							Key:                  "test-key",
						},
					},
				},
			},
		}
		// Execute
		err := controllerReconciler.fetchSecretsFromReferences(ctx, config)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key not found in secret")
	})
	t.Run("Successfully fetches secret and sets Value field", func(t *testing.T) {
		// Prepare
		fakeSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "some-secret"}, Data: map[string][]byte{"some-key": []byte("data")}}
		mockClient := runtimeclient.NewClientBuilder().WithObjects(fakeSecret).Build()
		controllerReconciler := ScalerConfigReconciler{
			Client: mockClient,
			Scheme: mockClient.Scheme(),
		}
		ctx := context.Background()

		config := globalConfig.DeepCopy()
		config.Spec.Config = v1alpha1.ScalerTypeConfigs{
			v1alpha1.RedisConfig{
				Host: "host",
				Port: "0",
				Password: v1alpha1.ValueOrSecret{
					Value: "",
					ValueFrom: v1alpha1.ValueSources{
						SecretKeyRef: corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "some-secret"},
							Key:                  "some-key",
						},
					},
				},
			},
		}
		// Execute
		err := controllerReconciler.fetchSecretsFromReferences(ctx, config)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "data", config.Spec.Config.Password.Value)

	})
}
