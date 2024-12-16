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

	"github.com/google/uuid"
	"github.com/quickube/QScaler/api/v1alpha1"
	"github.com/quickube/QScaler/internal/brokers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
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
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *QWorkerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	qworker := &v1alpha1.QWorker{}
	if err := r.Get(ctx, req.NamespacedName, qworker); err != nil {
		log.Log.Error(err, "unable to fetch QWorker")
		return ctrl.Result{Requeue: true}, err
	}

	var podList corev1.PodList
	if err := r.List(ctx, &podList, client.InNamespace(req.Namespace), client.MatchingFields{"metadata.ownerReferences.name": qworker.Name}); err != nil {
		return ctrl.Result{Requeue: true}, err
	}
	currentPodCount := len(podList.Items)
	qworker.Status.CurrentReplicas = currentPodCount

	if err := r.Status().Update(ctx, qworker); err != nil {
		log.Log.Error(err, fmt.Sprintf("Failed to update QWorker status %s", qworker.Name))
		return ctrl.Result{}, err
	}

	diffAmount := qworker.Status.DesiredReplicas - qworker.Status.CurrentReplicas
	log.Log.Info(fmt.Sprintf("going to deploy / takedown: %d pods", diffAmount))

		if diffAmount > 0 {
			for range diffAmount {
				if err := r.StartWorker(&ctx, qworker); err != nil {
					return ctrl.Result{Requeue: true}, err
				}
			}

		} else if diffAmount < 0 {
			for range diffAmount * -1 {
				if err := r.RemoveWorker(&ctx, qworker); err != nil {
					return ctrl.Result{Requeue: true}, err
				}
			}
		}


	log.Log.Info(fmt.Sprintf("Qworker %s replica count is %d", qworker.Name, qworker.Status.CurrentReplicas))
	if err := r.Status().Update(ctx, qworker); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *QWorkerReconciler) StartWorker(ctx *context.Context, qWorker *v1alpha1.QWorker) error {
	log.Log.Info("Starting worker", "name", qWorker.Name)
	podId := fmt.Sprintf("%s-%s", qWorker.ObjectMeta.Name, uuid.New().String())
	workerPod := &corev1.Pod{

		ObjectMeta: metav1.ObjectMeta{
			Name:      podId,
			Namespace: qWorker.ObjectMeta.Namespace,
		},
		Spec: qWorker.Spec.PodSpec,
	}

	// Set QWorker as the owner of the Pod
	if err := controllerutil.SetControllerReference(qWorker, workerPod, r.Scheme); err != nil {
		return err
	}

	if err := r.Create(*ctx, workerPod); err != nil {
		log.Log.Error(err, "unable to start worker pod")
		return err
	}
	qWorker.Status.CurrentReplicas += 1
	return nil
}
func (r *QWorkerReconciler) RemoveWorker(ctx *context.Context, qworker *v1alpha1.QWorker) error {

	BrokerClient, err := brokers.GetBroker(qworker.ObjectMeta.Namespace, qworker.Spec.ScaleConfig.ScalerConfigRef)
	if err != nil {
		log.Log.Error(err, "Failed to get broker client")
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
	// Add a field indexer for the ownerReferences.name field
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, "metadata.ownerReferences.name", func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		ownerRefs := pod.GetOwnerReferences()
		for _, ref := range ownerRefs {
			if ref.Kind == "QWorker" {
				return []string{ref.Name}
			}
		}
		return nil
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.QWorker{}).
		Watches(
			&corev1.Pod{},
			handler.EnqueueRequestForOwner(
				mgr.GetScheme(),
				mgr.GetRESTMapper(),
				&v1alpha1.QWorker{},
				handler.OnlyControllerOwner(), // Ensure we only enqueue for Pods controlled by QWorker
			),
		).
		Complete(r)
}
