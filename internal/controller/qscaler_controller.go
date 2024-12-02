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

	"github.com/google/uuid"
	"github.com/quickube/QScaler/api/v1alpha1"
	"github.com/quickube/QScaler/internal/brokers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// QWorkerReconciler reconciles a QScaler object
type QWorkerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=quickube.com,resources=qworkers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=quickube.com,resources=qworkers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=quickube.com,resources=qworkers/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create

func (r *QWorkerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	qworker := &v1alpha1.QWorker{}
	if err := r.Get(ctx, req.NamespacedName, qworker); err != nil {
		log.Log.Error(err, "unable to fetch QWorker")
		return ctrl.Result{}, err
	}

	// Fetch the ScalerConfig referenced in the QWorker
	var scalerConfig v1alpha1.ScalerConfig
	namespacedName := client.ObjectKey{Name: qworker.Spec.ScaleConfig.ScalerConfigRef, Namespace: qworker.ObjectMeta.Namespace}
	if err := r.Get(ctx, namespacedName, &scalerConfig); err != nil {
		log.Log.Error(err, "Failed to get ScalerConfig", "namespacedName", namespacedName.String())
		return ctrl.Result{}, err
	}

	BrokerClient, err := brokers.NewBroker(&scalerConfig)
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

	diffAmount := qworker.Status.DesiredReplicas - qworker.Status.CurrentReplicas
	log.Log.Info(fmt.Sprintf("going to deploy / takedown: %d pods", diffAmount))

	if diffAmount > 0 {
		for _ = range diffAmount {
			if err := r.StartWorker(&ctx, qworker); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else if diffAmount < 0 {
		for _ = range diffAmount * -1 {
			if err := r.RemoveWorker(&ctx, qworker); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	if err := r.Status().Update(ctx, qworker); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

func (r *QWorkerReconciler) StartWorker(ctx *context.Context, qWorker *v1alpha1.QWorker) error {
	podId := uuid.New().String()
	qWorker.ObjectMeta.Name = podId
	workerPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      qWorker.ObjectMeta.Name,
			Namespace: qWorker.ObjectMeta.Namespace,
		},
		Spec: qWorker.Spec.PodSpec,
	}
	if err := r.Create(*ctx, workerPod); err != nil {
		log.Log.Error(err, "unable to start worker pod")
		return err
	}
	qWorker.Status.CurrentReplicas += 1
	return nil
}
func (r *QWorkerReconciler) RemoveWorker(ctx *context.Context, qworker *v1alpha1.QWorker) error {

	var scalerConfig v1alpha1.ScalerConfig
	namespacedName := client.ObjectKey{Name: qworker.Spec.ScaleConfig.ScalerConfigRef, Namespace: qworker.ObjectMeta.Namespace}
	if err := r.Get(*ctx, namespacedName, &scalerConfig); err != nil {
		log.Log.Error(err, "Failed to get ScalerConfig", "namespacedName", namespacedName.String())
		return err
	}

	BrokerClient, err := brokers.NewBroker(&scalerConfig)
	if err != nil {
		log.Log.Error(err, "Failed to create broker client")
		return err
	}

	err = BrokerClient.KillQueue(ctx, qworker.Spec.ScaleConfig.Queue)
	if err != nil {
		log.Log.Error(err, "unable to kill queue")
		return err
	}
	qworker.Status.CurrentReplicas -= 1
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *QWorkerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.QWorker{}).
		Named("qscaler").
		Complete(r)
}
