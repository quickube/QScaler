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
	"time"

	"github.com/quickube/QScaler/api/v1alpha1"
	"github.com/quickube/QScaler/internal/brokers"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// MetricsControllerReconciler
type MetricsControllerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=quickube.com,resources=qworkers,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=quickube.com,resources=qworkers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=quickube.com,resources=qworkers/finalizers,verbs=update

// +kubebuilder:rbac:groups=quickube.com,resources=scalerconfigs,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=quickube.com,resources=scalerconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=quickube.com,resources=scalerconfigs/finalizers,verbs=update

func (r *MetricsControllerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	log.Log.Info("Metrics controller reconciling", "name", req.NamespacedName)

	qworkerList := &v1alpha1.QWorkerList{}

	err := r.List(context.Background(), qworkerList, &client.ListOptions{})
	if err != nil {
		log.Log.Error(err, "unable to list QWorker resources")
		return ctrl.Result{}, nil
	}

	// Iterate over the QWorker resources and trigger reconciliation for each one
	for _, qworker := range qworkerList.Items {
		// Create a reconcile request dynamically for each QWorker
		req := reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: qworker.Namespace,
				Name:      qworker.Name,
			},
		}

		qworker := &v1alpha1.QWorker{}
		if err := r.Get(ctx, req.NamespacedName, qworker); err != nil {
			log.Log.Error(err, "unable to fetch QWorker")
			return ctrl.Result{}, err
		}

		var scalerConfig v1alpha1.ScalerConfig
		namespacedName := client.ObjectKey{Name: qworker.Spec.ScaleConfig.ScalerConfigRef, Namespace: qworker.ObjectMeta.Namespace}
		if err := r.Get(ctx, namespacedName, &scalerConfig); err != nil {
			log.Log.Error(err, "Failed to get ScalerConfig", "namespacedName", namespacedName.String())
			return ctrl.Result{}, err
		}

		BrokerClient, err := brokers.GetBroker(req.Namespace, qworker.Spec.ScaleConfig.ScalerConfigRef)
		if err != nil {
			log.Log.Error(err, "Failed to create broker client")
			return ctrl.Result{}, err
		}

		QueueLength, err := BrokerClient.GetQueueLength(&ctx, qworker.Spec.ScaleConfig.Queue)
		if err != nil {
			log.Log.Error(err, "Failed to get queue length")
			return ctrl.Result{}, err
		}
		log.Log.Info(fmt.Sprintf("current queue length: %d", QueueLength))

		desiredPodsAmount := min(
			max(QueueLength*qworker.Spec.ScaleConfig.ScalingFactor, qworker.Spec.ScaleConfig.MinReplicas),
			qworker.Spec.ScaleConfig.MaxReplicas)
		log.Log.Info(fmt.Sprintf("desired amount: %d", desiredPodsAmount))
		qworker.Status.DesiredReplicas = desiredPodsAmount

		if err := r.Status().Update(ctx, qworker); err != nil {
			log.Log.Error(err, "Failed to update QWorker status")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MetricsControllerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ScalerConfig{}).
		Named("metrics").
		Complete(r)
}
