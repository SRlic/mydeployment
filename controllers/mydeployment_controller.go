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
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MyDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *MyDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	podList := &corev1.PodList{}
	instance := &mydeployment.MyDeployment{}

	//Init part
	//Get deployment and pods by client.get and client list
	_ = log.FromContext(ctx)
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			logr.Info("instance has been deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, nil
	}
	podGetErr := r.List(ctx, podList, client.InNamespace(req.Namespace), client.MatchingLabels{"app": instance.Name})
	if podGetErr != nil {
		logr.Error(podGetErr, "unable to get pod")
		return ctrl.Result{}, podGetErr
	}
	utils.StartLog(instance, logr)
	//init status
	statusPodList, initErr := InitCurrentStatus(ctx, instance, podList)
	if initErr != nil {
		logr.Error(initErr, "unable to init status")
		return ctrl.Result{}, podGetErr
	}

	//Core part
	//Start our process for deployment: scale or update podlist
	if instance.Status.CurrentOtherReplica == 0 {
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

	// TODO(user): your logic here
	err = r.Status().Update(ctx, instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	defer func() {
		utils.EndLog(instance, logr)
	}()
	return ctrl.Result{}, nil
}

//For each pod,  insert it into statusPodList and init our Deployment status
func InitCurrentStatus(ctx context.Context, deployment *mydeployment.MyDeployment, podList *corev1.PodList) (*utils.StatusPodList, error) {
	//record current status pod list
	statusPodList := utils.NewStatusPodList()
	deployment.Status = mydeployment.MyDeploymentStatus{}
	specImage := deployment.Spec.Image

	for i := range podList.Items {
		image := utils.GetImageStrFormPod(&podList.Items[i])
		if image == specImage {

			err := utils.ClassifyToMyPodList(ctx, statusPodList.SpecPod, &podList.Items[i])
			if err != nil {
				logr.Error(err, fmt.Sprintf("Init myDeployment error at ClassifyToPodList pod :%v", podList.Items[i]))
				return nil, err
			}
			//init deployment status
			deployment.Status.SpecPod.Add(mydeployment.NewDeployPod(&podList.Items[i]))
		} else {
			err := utils.ClassifyToMyPodList(ctx, statusPodList.OtherPod, &podList.Items[i])
			if err != nil {
				logr.Error(err, fmt.Sprintf("Init myDeployment error at ClassifyToPodList pod :%v", podList.Items[i]))
				return nil, err
			}
			//init deployment status
			deployment.Status.OtherPod.Add(mydeployment.NewDeployPod(&podList.Items[i]))
		}
	}
	//init deployment status
	deployment.Status.CurrentSpecReplica = deployment.Status.SpecPod.RunningReplica + deployment.Status.SpecPod.PendingReplica
	deployment.Status.CurrentOtherReplica = deployment.Status.OtherPod.RunningReplica + deployment.Status.OtherPod.PendingReplica
	deployment.Status.CurrentReplica = deployment.Status.CurrentSpecReplica + deployment.Status.CurrentOtherReplica
	//init StatusPodList
	statusPodList.CurrentSpecReplica = deployment.Status.CurrentSpecReplica
	statusPodList.CurrentOtherReplica = deployment.Status.CurrentOtherReplica
	logr.Info(fmt.Sprintf("current status: \r\n%v", deployment.Status))
	return statusPodList, nil
}

func (r *MyDeploymentReconciler) UpdatePod(ctx context.Context, podList *utils.StatusPodList, myDeployment *mydeployment.MyDeployment) error {
	//wait for pending spec pod
	if myDeployment.Status.SpecPod.PendingReplica > 0 {
		return nil
	}

	if myDeployment.Status.CurrentReplica == myDeployment.Spec.Replica {
		// roll update pod
		rollUpdateErr := r.RollUpdatePod(ctx, podList, myDeployment)
		if rollUpdateErr != nil {
			logr.Error(rollUpdateErr, "Roll update pod error")
			return rollUpdateErr
		}
	} else if myDeployment.Status.CurrentReplica > myDeployment.Spec.Replica {
		// running pod num is more than spec num
		deleteErr := r.BatchDeletePod(ctx, podList.OtherPod, myDeployment.Status.CurrentReplica-myDeployment.Spec.Replica)
		if deleteErr != nil {
			logr.Error(deleteErr, "Batch delete other pod error ")
			return deleteErr
		}
	} else {
		//running pod num is less than spec num
		createErr := r.BatchCreatePod(ctx, myDeployment, myDeployment.Spec.Replica-myDeployment.Status.CurrentReplica)
		if createErr != nil {
			logr.Error(createErr, "Batch create pod error")
			return createErr
		}
	}

	return nil
}

//Roll update our pod， Here we create one new pod，wait it util this pod is runing。 Delete one old pod
func (r *MyDeploymentReconciler) RollUpdatePod(ctx context.Context, podList *utils.StatusPodList, myDeployment *mydeployment.MyDeployment) error {
	//current running replica
	logr.Info(fmt.Sprintf("spec running replica %v, other running replica %v", podList.SpecPod.RunningReplica, podList.OtherPod.RunningReplica))
	runningReplica := podList.SpecPod.RunningReplica + podList.OtherPod.RunningReplica
	// new pod has runing, delete one pod other pod list
	if runningReplica > myDeployment.Spec.Replica {
		deleteErr := r.BatchDeletePod(ctx, podList.OtherPod, utils.BATCHUPDATESIZE)
		if deleteErr != nil {
			logr.Error(deleteErr, "Batch delete other pod error ")
			return deleteErr
		}

	} else {
		createErr := r.BatchCreatePod(ctx, myDeployment, utils.BATCHUPDATESIZE)
		if createErr != nil {
			logr.Error(createErr, "Batch create spec pod error ")
			return createErr
		}
	}
	return nil
}

// No other pod is running or pending, we do scale up / scale down for our pod list
func (r *MyDeploymentReconciler) ScalePod(ctx context.Context, podList *utils.StatusPodList, myDeployment *mydeployment.MyDeployment) error {
	//spec replica is bigger than current replica, scale up
	if myDeployment.Spec.Replica > myDeployment.Status.CurrentSpecReplica {
		//Create New Pod
		size := myDeployment.Spec.Replica - myDeployment.Status.CurrentSpecReplica
		createErr := r.BatchCreatePod(ctx, myDeployment, size)
		if createErr != nil {
			logr.Error(createErr, "Batch create pod error")
			return createErr
		}
	} else if myDeployment.Spec.Replica < myDeployment.Status.CurrentReplica {
		//spec replica is less than current replica, scale down
		deleteNum := myDeployment.Status.CurrentSpecReplica - myDeployment.Spec.Replica
		deleteErr := r.BatchDeletePod(ctx, podList.SpecPod, deleteNum)
		if deleteErr != nil {
			logr.Error(deleteErr, "Batch delete pod error ")
			return deleteErr
		}
	} else {
		//some pods transfer from pending to running or some pods are Succeeded/Failed
		return nil
	}
	return nil
}

//Batch Create Pod
func (r *MyDeploymentReconciler) BatchCreatePod(ctx context.Context, myDeployment *mydeployment.MyDeployment, size int) error {
	logr.Info(fmt.Sprintf("create pod num: %v, image:%v", size, myDeployment.Spec.Image))
	rand.Seed(time.Now().Unix())
	// podList := make([]*corev1.Pod, 0)
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
		_, createErr := ctrl.CreateOrUpdate(ctx, r.Client, newPod, func() error {
			return ctrl.SetControllerReference(myDeployment, newPod, r.Scheme)
		})
		if createErr != nil {
			logr.Error(createErr, "Create pod error")
			return createErr
		}

	}
	return nil
}

// Delete pod from my pod list, first delete pending pod , then delete running pod
func (r *MyDeploymentReconciler) BatchDeletePod(ctx context.Context, podList *utils.MyPodList, deleteNum int) error {
	logr.Info(fmt.Sprintf("delete pod num: %v, ", deleteNum))
	pendingLen := podList.PendingReplica
	runningLen := podList.RunningReplica
	//Delete pending pod first
	i := 0
	for i = 0; i < deleteNum && i < pendingLen; i++ {
		logr.Info(fmt.Sprintf("delete pod : %v, ", podList.PendingPod[i].Name))
		if err := r.Delete(ctx, podList.PendingPod[i]); err != nil {
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
		logr.Info(fmt.Sprintf("delete pod : %v, ", podList.RunningPod[i].Name))
		if err := r.Delete(ctx, podList.RunningPod[i]); err != nil {
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
