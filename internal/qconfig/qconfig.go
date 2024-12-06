package qconfig

import (
	"context"
	"fmt"
	"github.com/quickube/QScaler/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func UpdateConfigPasswordValue(ctx context.Context, client runtimeclient.Client, config *v1alpha1.ScalerConfig) error {
	_ = log.FromContext(ctx)

	if config.Spec.Config.Password.Value == "" {
		secretRef := config.Spec.Config.Password.ValueFrom.SecretKeyRef

		passwordSecret := &corev1.Secret{}

		namespacedName := types.NamespacedName{Namespace: config.Namespace, Name: secretRef.Name}
		if err := client.Get(ctx, namespacedName, passwordSecret); err != nil {
			log.Log.Error(err, fmt.Sprintf("unable to fetch config %s", namespacedName))
			return err
		}
		actualPassword, exists := passwordSecret.Data[secretRef.Key]
		if !exists {
			return fmt.Errorf("key %s not found in secret %s", secretRef.Key, secretRef.Name)
		}

		config.Spec.Config.Password.Value = string(actualPassword)
	}
	return nil
}
