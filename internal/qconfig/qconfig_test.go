package qconfig_test

import (
	"context"
	"github.com/quickube/QScaler/internal/qconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	"github.com/quickube/QScaler/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/assert"
)

func TestFetchSecretsFromReferences(t *testing.T) {
	globalConfig := &v1alpha1.ScalerConfig{
		Spec: v1alpha1.ScalerConfigSpec{
			Type:   "test-type",
			Config: v1alpha1.ScalerTypeConfigs{}}}

	t.Run("Fail on Secret not found", func(t *testing.T) {
		// Prepare
		mockClient := runtimeclient.NewFakeClient()
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
		err := qconfig.FetchSecretsFromReferences(ctx, mockClient, config)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, err.Error(), "secrets \"missing-secret\" not found")
	})
	t.Run("Fail on key not in secret", func(t *testing.T) {
		// Prepare
		fakeSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "some-secret"}, Data: map[string][]byte{"other-key": {}}}
		mockClient := runtimeclient.NewClientBuilder().WithObjects(fakeSecret).Build()
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
		err := qconfig.FetchSecretsFromReferences(ctx, mockClient, config)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key not found in secret")
	})
	t.Run("Successfully fetches secret and sets Value field", func(t *testing.T) {
		// Prepare
		fakeSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "some-secret"}, Data: map[string][]byte{"some-key": []byte("data")}}
		mockClient := runtimeclient.NewClientBuilder().WithObjects(fakeSecret).Build()
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
		err := qconfig.FetchSecretsFromReferences(ctx, mockClient, config)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "data", config.Spec.Config.Password.Value)

	})
}
