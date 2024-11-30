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
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	v1 "github.com/quickube/QScale/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"math"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

// QScaleReconciler reconciles a QScale object
type QScaleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=github.com,resources=qworkers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=github.com,resources=qworkers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=github.com,resources=qworkers/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=pods/finalizers,verbs=update

func (r *QScaleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	scaler := &v1.QWorker{}
	if err := r.Get(ctx, req.NamespacedName, scaler); err != nil {
		log.Log.Error(err, "unable to fetch QWorker")
		return ctrl.Result{}, err
	}
	redisClient := GetRedisClient(scaler, &ctx)
	QLen, err := GetQueueLength(redisClient, scaler, &ctx)
	if err != nil {
		return ctrl.Result{}, err
	}
	log.Log.Info(fmt.Sprintf("current queue length: %d", QLen))

	desiredPodsAmount := min(max(QLen*scaler.Spec.ScaleConfig.ScalingFactor, scaler.Spec.ScaleConfig.MinReplicas), scaler.Spec.ScaleConfig.MaxReplicas)
	log.Log.Info(fmt.Sprintf("desired amount: %d", desiredPodsAmount))
	scaler.Status.DesiredReplicas = desiredPodsAmount

	diffAmount := int64(math.Abs(float64(scaler.Status.DesiredReplicas - scaler.Status.CurrentReplicas)))
	log.Log.Info(fmt.Sprintf("going to deploy / takedown: %d pods", diffAmount))

	if scaler.Status.DesiredReplicas > scaler.Status.CurrentReplicas {
		for _ = range diffAmount {
			if err := r.StartWorker(scaler, &ctx); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		for _ = range diffAmount {
			if err := r.RemoveWorker(scaler, redisClient, &ctx); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	if err := r.Status().Update(ctx, scaler); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

func (r *QScaleReconciler) StartWorker(scaler *v1.QWorker, ctx *context.Context) error {
	podId := uuid.New().String()
	scaler.Spec.ObjectMeta.Name = fmt.Sprintf("%s-%s-%s", scaler.Spec.ObjectMeta.Name, scaler.Spec.ScaleConfig.Queue, podId)
	workerPod := &corev1.Pod{
		ObjectMeta: scaler.Spec.ObjectMeta,
		Spec:       scaler.Spec.PodSpec,
	}
	if err := r.Create(*ctx, workerPod); err != nil {
		log.Log.Error(err, "unable to start worker pod")
		return err
	}
	scaler.Status.CurrentReplicas += 1
	return nil
}
func (r *QScaleReconciler) RemoveWorker(scaler *v1.QWorker, redisClient *redis.Client, ctx *context.Context) error {
	status := redisClient.LPush(*ctx, scaler.GetDeathQueue(), "{'kill': 'true'}")
	if status.String() == "error" {
		return errors.New(status.String())
	}
	log.Log.Info(fmt.Sprintf("published message to death queue: %s", status))
	scaler.Status.CurrentReplicas -= 1
	return nil
}

func GetRedisClient(scaler *v1.QWorker, ctx *context.Context) *redis.Client {
	Rclient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", scaler.Spec.ScaleConfig.BrokerConfig.Host, scaler.Spec.ScaleConfig.BrokerConfig.Port),
		Password: scaler.Spec.ScaleConfig.BrokerConfig.Password,
	})
	redisStatus := Rclient.Ping(*ctx)
	log.Log.Info(fmt.Sprintf("Redis Status: %v", redisStatus))
	return Rclient
}

func GetQueueLength(redisClient *redis.Client, scaler *v1.QWorker, ctx *context.Context) (int64, error) {
	taskQueueLength, err := redisClient.LLen(*ctx, scaler.Spec.ScaleConfig.Queue).Result()
	if err != nil {
		return -1, err
	}
	return taskQueueLength, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *QScaleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.QWorker{}).
		Named("qscale").
		Complete(r)
}
