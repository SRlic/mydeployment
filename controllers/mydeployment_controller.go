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
	logr.Info("Reconcile start =====================")

	// 1. Get deployment, list pods owned by this deployment
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			logr.Info("Deployment has been deleted")
			return ctrl.Result{}, nil
		}
		logr.Error(err, "Get Deployment failed")
		return ctrl.Result{}, err
	}
	logr.Info(fmt.Sprintf("Deployment spec image:%v , replica: %v ", instance.Spec.Image, instance.Spec.Replica))

	err = r.List(ctx, podList, client.InNamespace(req.Namespace), client.MatchingLabels{"app": instance.Name})
	if err != nil {
		if errors.IsNotFound(err) {
			logr.Info("All pods has been deleted")
			return ctrl.Result{}, nil
		}
		logr.Error(err, "List pod failed")
		return ctrl.Result{}, err
	}

	// 2.Initialize the StatusPodList.
	// In StatusPodList, pods are divided into two categories: specpod and otherpod,
	// which are inserted into specpodlist and otherpodlist respectively
	statusPodList, err := utils.NewStatusPodList(podList, instance.Spec.Image)
	if err != nil {
		logr.Error(err, "Get new status pod list failed ")
		return ctrl.Result{}, err
	}

	// 3. Core part，process pod scaling or updating
	// If the num of other pods is 0，process pod scaling up or down
	// else process pod updating
	if statusPodList.CurrentOtherReplica == 0 {
		scaleErr := r.ScalePod(ctx, statusPodList, instance)
		if scaleErr != nil {
			logr.Error(scaleErr, "scale pod error ")
			return ctrl.Result{}, err
		}
	} else {
		udpateErr := r.UpdatePod(ctx, statusPodList, instance)
		if udpateErr != nil {
			logr.Error(udpateErr, "update pod error ")
			return ctrl.Result{}, err
		}
	}

	// 4. Update deployment status
	err = r.UpdateDeploymentStatus(ctx, statusPodList, instance)
	if err != nil {
		logr.Error(err, "Update deployment status failed ")
		return ctrl.Result{}, err
	}

	defer func() {
		logr.Info("=====================  Reconcile end")
	}()
	return ctrl.Result{}, nil
}

// Convert StatusPodList into DeploymentStatus, get current deployemnt status phase and update the status subresource of current mydeployment
func (r *MyDeploymentReconciler) UpdateDeploymentStatus(ctx context.Context, statusPodList *utils.StatusPodList, myDeployment *mydeployment.MyDeployment) error {
	myDeployment.Status = *statusPodList.ToDeploymentStatus()
	logr.Info(fmt.Sprintf("myDeployment Status %v", myDeployment.Status))

	myDeployment.Status.UpdateStatusPhase(&myDeployment.Spec)
	logr.Info(fmt.Sprintf("My Depolyment state %v", myDeployment.Status.Phase))

	err := r.Status().Update(ctx, myDeployment)
	if err != nil {
		return err
	}
	return nil
}

// UpdatePod handles the pod updating process
func (r *MyDeploymentReconciler) UpdatePod(ctx context.Context, statusPodList *utils.StatusPodList, myDeployment *mydeployment.MyDeployment) error {
	logr.Info("Update pod start")
	defer func() {
		logr.Info("Update pod end")
	}()
	// Wait for pending spec pods
	if statusPodList.SpecPodList.PendingReplica > 0 {
		logr.Info("Waiting for some pending specPods")
		return nil
	}

	// If there are pending other pods, delete them directly
	if statusPodList.OtherPodList.PendingReplica > 0 {
		logr.Info("Delete pending otherPods")
		err := r.BatchDeletePod(ctx, statusPodList.OtherPodList, statusPodList.OtherPodList.PendingReplica)
		if err != nil {
			logr.Error(err, "Batch delete pod error ")
			return err
		}
		return nil
	}

	// No pending pod, roll update pod
	return r.RollUpdatePod(ctx, statusPodList, myDeployment)
}

// RollUpdatePod handles the pod roll updating, the rolling update granularity is set to 1 here
func (r *MyDeploymentReconciler) RollUpdatePod(ctx context.Context, statusPodList *utils.StatusPodList, myDeployment *mydeployment.MyDeployment) error {
	logr.Info("Roll update pod")
	logr.Info(fmt.Sprintf("spec running replica %v, other running replica %v", statusPodList.SpecPodList.RunningReplica, statusPodList.OtherPodList.RunningReplica))
	runningReplica := statusPodList.SpecPodList.RunningReplica + statusPodList.OtherPodList.RunningReplica

	if runningReplica > myDeployment.Spec.Replica {
		err := r.BatchDeletePod(ctx, statusPodList.OtherPodList, utils.BATCHUPDATESIZE)
		if err != nil {
			logr.Error(err, "Batch delete other pod error ")
			return err
		}
	} else {
		err := r.BatchCreatePod(ctx, myDeployment, utils.BATCHUPDATESIZE)
		if err != nil {
			logr.Error(err, "Batch create spec pod error ")
			return err
		}
	}
	return nil
}

// ScalePod handles the pod scaling process
func (r *MyDeploymentReconciler) ScalePod(ctx context.Context, statusPodList *utils.StatusPodList, myDeployment *mydeployment.MyDeployment) error {
	logr.Info("Scale pod start")
	defer func() {
		logr.Info("Scale pod end")
	}()

	specReplica := myDeployment.Spec.Replica
	currentReplica := statusPodList.CurrentSpecReplica

	if specReplica > currentReplica {
		// Spec replica is larger than current replica, scale up
		createNum := specReplica - currentReplica
		err := r.BatchCreatePod(ctx, myDeployment, createNum)
		if err != nil {
			logr.Error(err, "Batch create pod error")
			return err
		}
	} else if specReplica < currentReplica {
		// Spec replica is less than current replica, scale down
		deleteNum := myDeployment.Status.CurrentSpecReplica - myDeployment.Spec.Replica
		err := r.BatchDeletePod(ctx, statusPodList.SpecPodList, deleteNum)
		if err != nil {
			logr.Error(err, "Batch delete pod error ")
			return err
		}
	} else {
		// Spec replica is equal with current replica, do nothing
	}

	return nil
}

// Batch Create Pod
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

// Delete pod from my pod list, pending pods will be deleted first
func (r *MyDeploymentReconciler) BatchDeletePod(ctx context.Context, podList *utils.MyPodList, deleteNum int) error {
	logr.Info(fmt.Sprintf("delete pod num: %v, ", deleteNum))
	pendingLen := podList.PendingReplica
	runningLen := podList.RunningReplica
	//Delete pending pod first
	i := 0
	for i = 0; i < deleteNum && i < pendingLen; i++ {
		logr.Info(fmt.Sprintf("delete pod : %v, ", podList.PendingPods[i].Name))
		if err := r.Delete(ctx, podList.PendingPods[i]); err != nil {
			if errors.IsNotFound(err) {
				logr.Info("pod has been deleted")
				return nil
			}
			logr.Error(err, "delete pod error")
			return err
		}

	}
	deleteNum = deleteNum - i
	//Delete runing pod then
	for i = 0; i < deleteNum && i < runningLen; i++ {
		logr.Info(fmt.Sprintf("delete pod : %v, ", podList.RunningPods[i].Name))
		if err := r.Delete(ctx, podList.RunningPods[i]); err != nil {
			if errors.IsNotFound(err) {
				logr.Info("pod has been deleted")
				return nil
			}
			logr.Error(err, "delete pod error")
			return err
		}
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
