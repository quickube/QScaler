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
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ScalerConfigReconciler reconciles a ScalerConfig object
type ScalerConfigReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=quickube.com,resources=scalerconfigs,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=quickube.com,resources=scalerconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=quickube.com,resources=scalerconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;

func (r *ScalerConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var err error
	var ok bool

	reqLogger := log.FromContext(ctx)
	reqLogger.Info(fmt.Sprintf("reconcileing Scaler: %s", req.Name))

	scalerConfig := &v1alpha1.ScalerConfig{}
	if err = r.Get(ctx, req.NamespacedName, scalerConfig); err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		reqLogger.Error(err, fmt.Sprintf("unable to fetch ScalerConfig %s", req.NamespacedName))
		return ctrl.Result{}, err
	}

	broker, err := brokers.NewBroker(scalerConfig)
	if err != nil {
		r.Recorder.Eventf(scalerConfig, corev1.EventTypeWarning, "FailedToCreateBroker", err.Error())
		reqLogger.Error(err, fmt.Sprintf("unable to create broker %s", req.NamespacedName))
		_ = r.updateScalerHealth(&ctx, scalerConfig, false)
		return ctrl.Result{}, err
	}

	r.Recorder.Eventf(scalerConfig, corev1.EventTypeNormal, "Broker", "Broker %s initilizaed", scalerConfig.Spec.Type)

	if ok, err = broker.IsConnected(&ctx); !ok || err != nil {
		reqLogger.Error(err, "Failed to connect to broker", "name", req.NamespacedName)
		r.Recorder.Eventf(scalerConfig, corev1.EventTypeWarning, "Broker", "Failed to connect to broker %s", req.NamespacedName)
		_ = r.updateScalerHealth(&ctx, scalerConfig, false)
		return ctrl.Result{}, err
	}

	r.Recorder.Eventf(scalerConfig, corev1.EventTypeNormal, "Broker", "Broker %s created suceffully ", scalerConfig.Spec.Type)
	reqLogger.Info("ScalerConfig reconciled", "name", req.NamespacedName)
	err = r.updateScalerHealth(&ctx, scalerConfig, true)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ScalerConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ScalerConfig{}).
		Watches(
			&corev1.Secret{}, // Watch Secret resources
			handler.EnqueueRequestsFromMapFunc(r.SecretToScalerConfigMapFunc())).
		Complete(r)
}

func (r *ScalerConfigReconciler) updateScalerHealth(ctx *context.Context, scalerConfig *v1alpha1.ScalerConfig, health bool) error {
	scalerConfig.Status.Healthy = health
	log.Log.Info("Updating ScalerConfig", "name", scalerConfig.Name, "health", scalerConfig.Status.Healthy)
	if err := r.Status().Update(*ctx, scalerConfig); err != nil {
		log.Log.Error(err, "Failed to update scalerConfig status", "name", scalerConfig.Name)
		return err
	}
	return nil
}

func (r *ScalerConfigReconciler) SecretToScalerConfigMapFunc() func(context.Context, client.Object) []reconcile.Request {
	return func(ctx context.Context, obj client.Object) []reconcile.Request {
		secret, ok := obj.(*corev1.Secret)
		if !ok {
			return nil // Not a Secret, ignore.
		}

		// List all ScalerConfig resources in the same namespace as the Secret
		scalerConfigList := &v1alpha1.ScalerConfigList{}
		if err := r.List(ctx, scalerConfigList, client.InNamespace(secret.Namespace)); err != nil {
			return nil // Return an empty list if the listing fails
		}

		// Check which ScalerConfig references this Secret
		var requests []reconcile.Request
		for _, scalerConfig := range scalerConfigList.Items {
			if scalerConfig.ReferencesSecret(secret.Name) {
				log.Log.Info("reconsiling due to secret change", "name", scalerConfig.Name)
				// Enqueue a reconcile request for the ScalerConfig
				requests = append(requests, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Name:      scalerConfig.Name,
						Namespace: scalerConfig.Namespace,
					},
				})
			}
		}

		return requests
	}
}
