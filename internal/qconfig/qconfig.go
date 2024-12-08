package qconfig

import (
	"context"
	"fmt"
	"reflect"

	"github.com/quickube/QScaler/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func FetchSecretsFromReferences(ctx context.Context, client runtimeclient.Client, config *v1alpha1.ScalerConfig) error {
	_ = log.FromContext(ctx)

	v := reflect.ValueOf(&config.Spec.Config).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		for x := 0; x < field.NumField(); x++ {
			configField := field.Field(x)
			if configField.Type() == reflect.TypeOf(v1alpha1.ValueOrSecret{}) {
				valueOrSecret := configField.Interface().(v1alpha1.ValueOrSecret)
				if valueOrSecret.Value == "" {
					secretRef := valueOrSecret.ValueFrom.SecretKeyRef
					actualSecret := &corev1.Secret{}
					namespacedName := types.NamespacedName{Namespace: config.Namespace, Name: secretRef.Name}
					if err := client.Get(ctx, namespacedName, actualSecret); err != nil {
						return err
					}
					secretData, exists := actualSecret.Data[secretRef.Key]
					if !exists {
						return fmt.Errorf("key not found in secret:  %s.%s", secretRef.Name, secretRef.Key)
					}

					valueOrSecret.Value = string(secretData)
					configField.Set(reflect.ValueOf(valueOrSecret))
				}
			}

		}
	}
	return nil
}
