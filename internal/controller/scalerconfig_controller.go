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
	"fmt"
	"github.com/quickube/QScaler/api/v1alpha1"
	"github.com/quickube/QScaler/internal/brokers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ScalerConfigReconciler reconciles a ScalerConfig object
type ScalerConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=quickube.com,resources=scalerconfigs,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=quickube.com,resources=scalerconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=quickube.com,resources=scalerconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *ScalerConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	log.Log.Info(fmt.Sprintf("got request for: %+v", req.NamespacedName))

	scalerConfig := &v1alpha1.ScalerConfig{}
	if err := r.Get(ctx, req.NamespacedName, scalerConfig); err != nil {
		if errors.IsNotFound(err) {
			log.Log.Info("ScaleConfig resource not found")
			return ctrl.Result{}, nil
		}
		log.Log.Error(err, fmt.Sprintf("unable to fetch ScalerConfig %s", req.NamespacedName))
		return ctrl.Result{Requeue: true}, err
	}

	if err := r.fetchSecretsFromReferences(ctx, scalerConfig); err != nil {
		// removing broker as config might have changed
		brokers.RemoveBroker(scalerConfig.Namespace, scalerConfig.Name)
		return ctrl.Result{Requeue: true}, err
	}

	broker, err := brokers.NewBroker(scalerConfig)
	if err != nil {
		log.Log.Error(err, fmt.Sprintf("unable to create broker %s", req.NamespacedName))
		return ctrl.Result{}, err
	}

	if ok, err := broker.IsConnected(&ctx); !ok || err != nil {
		scalerConfig.Status.Healthy = false
		scalerConfig.Status.Message = "Failed to connect to broker"
		log.Log.Error(err, "Failed to connect to broker", "name", req.NamespacedName)
		if err = r.Status().Update(ctx, scalerConfig); err != nil {
			log.Log.Error(err, "Failed to update scalerConfig status", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	} else {
		scalerConfig.Status.Healthy = true
		scalerConfig.Status.Message = "Connected to broker"
		if err = r.Status().Update(ctx, scalerConfig); err != nil {
			log.Log.Error(err, "Failed to update scalerConfig status", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
	}

	log.Log.Info("ScalerConfig reconciled", "name", req.NamespacedName)
	return ctrl.Result{}, nil
}

func (r *ScalerConfigReconciler) fetchSecretsFromReferences(ctx context.Context, config *v1alpha1.ScalerConfig) error {
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
					if err := r.Get(ctx, namespacedName, actualSecret); err != nil {
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

// SetupWithManager sets up the controller with the Manager.
func (r *ScalerConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ScalerConfig{}).
		Named("scalerconfig").
		Complete(r)
}
