/*
Copyright 2022.

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

package controllers

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mydeployment "mydeployment/api/v1"
	"mydeployment/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MyDeploymentReconciler reconciles a MyDeployment object
type MyDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	RollUpgradeGranularity = 1
)

var (
	logr = log.Log.WithName("myDeployment")
)

//+kubebuilder:rbac:groups=kubelearn.liyichen.kubebuilder.io,resources=mydeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kubelearn.liyichen.kubebuilder.io,resources=mydeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kubelearn.liyichen.kubebuilder.io,resources=mydeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *MyDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	podList := &corev1.PodList{}
	instance := &mydeployment.MyDeployment{}
	logr.Info("Reconcile start")
	defer func() {
		logr.Info("Reconcile end")
	}()

	// Get deployment, list pods owned by this deployment
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			logr.Info("Deployment not found")
			return ctrl.Result{}, nil
		}
		logr.Error(err, "Get Deployment failed")
		return ctrl.Result{}, err
	}
	logr.Info(fmt.Sprintf("Deployment spec image:%v , replica: %v ", instance.Spec.Image, instance.Spec.Replica))

	err = r.List(ctx, podList, client.InNamespace(req.Namespace), client.MatchingLabels{"app": instance.Name})
	if err != nil {
		if errors.IsNotFound(err) {
			logr.Info(fmt.Sprintf("Pods owns by deployment %v not found", instance.Name))
			return ctrl.Result{}, nil
		}
		logr.Error(err, "List pod failed")
		return ctrl.Result{}, err
	}

	specPodNum, expiredPodNum := r.getPodStatistics(podList, instance)

	if expiredPodNum != 0 {
		// need upgrade
		err := r.ProcessUpgrade(ctx, podList, instance)
		if err != nil {
			logr.Error(err, "pod upgrade error")
			return ctrl.Result{}, nil
		}

	} else if specPodNum != instance.Spec.Replica {
		// need scale
		err := r.ProcessScale(ctx, podList, instance, specPodNum)
		if err != nil {
			logr.Error(err, "pod scale error")
			return ctrl.Result{}, nil
		}
	} else {
		err := r.updateDeploymentStatus(ctx, instance, mydeployment.DepolyRuning)
		if err != nil {
			logr.Error(err, "update deployment status error")
			return ctrl.Result{}, err
		}
	}

	r.updateDeploymentStatus(ctx, instance, mydeployment.DepolyRuning)

	return ctrl.Result{}, nil
}
func (r *MyDeploymentReconciler) ProcessScale(ctx context.Context, podList *corev1.PodList, myDeployment *mydeployment.MyDeployment, specPodNum int) error {
	err := r.updateDeploymentStatus(ctx, myDeployment, mydeployment.DeployScaling)
	if err != nil {
		logr.Error(err, "update deployment status error")
		return err
	}

	if specPodNum > myDeployment.Spec.Replica {
		// scale down
		err := r.BatchDeletePod(ctx, podList, myDeployment.Spec.Replica-specPodNum)
		if err != nil {
			logr.Error(err, "Batch delete pod error ")
			return err
		}
	} else {
		// scale up
		err := r.BatchCreatePod(ctx, myDeployment, specPodNum-myDeployment.Spec.Replica)
		if err != nil {
			logr.Error(err, "Batch create pod error ")
			return err
		}
	}

	return nil
}

func (r *MyDeploymentReconciler) ProcessUpgrade(ctx context.Context, podList *corev1.PodList, myDeployment *mydeployment.MyDeployment) error {
	err := r.updateDeploymentStatus(ctx, myDeployment, mydeployment.DepolyUpgrating)
	if err != nil {
		logr.Error(err, "update deployment status error")
		return err
	}

	if r.needWaitForPendingSpecPod(podList, myDeployment.Spec.Image) {
		return nil
	}

	r.deletePendingExpiredPod(ctx, podList, myDeployment.Spec.Image)

	return r.ProcessRollUpgrade(ctx, podList, myDeployment)
}

func (r *MyDeploymentReconciler) ProcessRollUpgrade(ctx context.Context, podList *corev1.PodList, myDeployment *mydeployment.MyDeployment) error {
	runningPodNum := 0
	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodRunning && pod.DeletionTimestamp.IsZero() {
			runningPodNum++
		}
	}

	if runningPodNum > myDeployment.Spec.Replica {
		err := r.BatchDeleteExpiredPod(ctx, podList, myDeployment.Spec.Image, myDeployment.Spec.Replica-runningPodNum)
		if err != nil {
			logr.Error(err, "Batch delete expired pod error ")
		}
	} else if runningPodNum == myDeployment.Spec.Replica {
		err := r.BatchCreatePod(ctx, myDeployment, RollUpgradeGranularity)
		if err != nil {
			logr.Error(err, "Batch create pod error ")
		}

	} else {
		err := r.BatchCreatePod(ctx, myDeployment, runningPodNum-myDeployment.Spec.Replica)
		if err != nil {
			logr.Error(err, "Batch create pod error ")
		}
	}
	return nil
}

func (r *MyDeploymentReconciler) needWaitForPendingSpecPod(podList *corev1.PodList, image string) bool {
	for _, pod := range podList.Items {
		if utils.IsSpecPod(&pod, image) && pod.Status.Phase == corev1.PodPending {
			return true
		}
	}
	return false
}

func (r *MyDeploymentReconciler) deletePendingExpiredPod(ctx context.Context, podList *corev1.PodList, image string) error {
	for _, pod := range podList.Items {
		if !utils.IsSpecPod(&pod, image) && pod.Status.Phase == corev1.PodPending {
			if pod.DeletionTimestamp.IsZero() {
				err := r.Client.Delete(ctx, &pod)
				if err != nil {
					logr.Error(err, fmt.Sprintf("Delete pod: %v error", pod))
					return err
				}
			}
		}
	}
	return nil
}

func (r *MyDeploymentReconciler) getPodStatistics(podList *corev1.PodList, myDeployment *mydeployment.MyDeployment) (int, int) {
	var (
		specPodNum    int = 0
		expiredPodNum int = 0
	)

	for _, pod := range podList.Items {
		if !pod.DeletionTimestamp.IsZero() {
			continue
		}
		if myDeployment.Spec.Image == utils.GetImageStrFromPod(&pod) {
			specPodNum++
		} else {
			expiredPodNum++
		}
	}
	return specPodNum, expiredPodNum
}

// UpdateDeploymentStatus Convert StatusPodList into DeploymentStatus, get current deployemnt status phase and update the status subresource of current mydeployment
func (r *MyDeploymentReconciler) UpdateDeploymentStatus(ctx context.Context, statusPodList *utils.StatusPodList, myDeployment *mydeployment.MyDeployment) error {
	status := *statusPodList.ToDeploymentStatus()
	logr.Info(fmt.Sprintf("myDeployment Status %v", status))

	status.UpdateStatusPhase(&myDeployment.Spec)
	logr.Info(fmt.Sprintf("My Depolyment state %v", status.Phase))

	if !reflect.DeepEqual(myDeployment.Status, status) {
		myDeployment.Status = status
		err := r.Status().Update(ctx, myDeployment)
		if err != nil {
			logr.Error(err, "Failed to update myDeployment status")
			return err
		}
	}

	return nil
}

// BatchCreatePod Batch Create Pod
func (r *MyDeploymentReconciler) BatchCreatePod(ctx context.Context, myDeployment *mydeployment.MyDeployment, size int) error {
	logr.Info(fmt.Sprintf("create pod num: %v, image:%v", size, myDeployment.Spec.Image))
	rand.Seed(time.Now().Unix())

	for i := 0; i < size; i++ {
		newPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      myDeployment.Name + myDeployment.Spec.Image + utils.RandStr(utils.PODNAMESIZE),
				Namespace: myDeployment.Namespace,
				Labels: map[string]string{
					"app": myDeployment.Name,
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  myDeployment.Spec.Image,
						Image: myDeployment.Spec.Image,
					},
				},
				RestartPolicy: corev1.RestartPolicyOnFailure,
			},
		}

		logr.Info(fmt.Sprintf("create pod : %v, ", newPod.Name))

		_, err := ctrl.CreateOrUpdate(ctx, r.Client, newPod, func() error {
			return ctrl.SetControllerReference(myDeployment, newPod, r.Scheme)
		})
		if err != nil {
			logr.Error(err, "Create pod error")
			return err
		}

	}
	return nil
}

// BatchDeletePod Delete pod from pod list
func (r *MyDeploymentReconciler) BatchDeletePod(ctx context.Context, podList *corev1.PodList, deleteNum int) error {
	logr.Info(fmt.Sprintf("delete pod num: %v, ", deleteNum))
	for _, pod := range podList.Items {
		if deleteNum <= 0 {
			break
		}
		if pod.DeletionTimestamp.IsZero() {
			err := r.Client.Delete(ctx, &pod)
			if err != nil {
				logr.Error(err, fmt.Sprintf("Delete pod: %v error", pod))
				return err
			}
			deleteNum--
		}

	}
	return nil

}

// BatchDeletePod Delete pod from pod list
func (r *MyDeploymentReconciler) BatchDeleteExpiredPod(ctx context.Context, podList *corev1.PodList, image string, deleteNum int) error {
	logr.Info(fmt.Sprintf("delete pod num: %v, ", deleteNum))
	for _, pod := range podList.Items {
		if deleteNum <= 0 {
			break
		}
		if pod.DeletionTimestamp.IsZero() && image != utils.GetImageStrFromPod(&pod) {
			err := r.Client.Delete(ctx, &pod)
			if err != nil {
				logr.Error(err, fmt.Sprintf("Delete pod: %v error", pod))
				return err
			}
			deleteNum--
		}

	}
	return nil

}
func (r *MyDeploymentReconciler) updateDeploymentStatus(ctx context.Context, myDeployment *mydeployment.MyDeployment, currentPhase string) error {
	myDeployment.Status.Phase = currentPhase
	err := r.Status().Update(ctx, myDeployment)
	if err != nil {
		logr.Error(err, "Failed to update myDeployment status")
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mydeployment.MyDeployment{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
