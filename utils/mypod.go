package utils

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
)

//Classfy the pod into running pod, pending pod and delete pod
//If the DeletionTimestamp in pod is not nil, the pod is deleted, we ignore whether pod is succeed of failed
//If the status phase in pod is pending->PendingPod
//If the status phase in pod is running->RunningPod
//When we need delete the pod, we first delete the pending pods
type MyPodList struct {
	RunningReplica int `json:"running_replica,omitempty"`
	PendingReplica int `json:"pedding_replica,omitempty"`
	DeleteReplica  int `json:"delete_replica,omitempty"`

	RunningPod []*corev1.Pod `json:"running_pod,omitempty"`
	PendingPod []*corev1.Pod `json:"pending_pod,omitempty"`
	DeletePod  []*corev1.Pod `json:"delete_pod,omitempty"`
}

func NewMyPodList() *MyPodList {
	return &MyPodList{
		RunningPod: make([]*corev1.Pod, 0),
		PendingPod: make([]*corev1.Pod, 0),
		DeletePod:  make([]*corev1.Pod, 0),
	}
}

//StatusPodList record the pods in current deployment.
//The pods is classified into SpecPod and OtherPod,
//SpecPod->pod->image == spec image
//OtherPod->pod->image != spec image
type StatusPodList struct {
	CurrentSpecReplica int        `json:"current_spec_replica,omitempty"`
	SpecPod            *MyPodList `json:"spec_pod,omitempty"`

	//current other replica= OtherPod.RunningReplica + OtherPod.PendingReplica
	CurrentOtherReplica int        `json:"current_other_replica,omitempty"`
	OtherPod            *MyPodList `json:"other_pod,omitempty"`
}

func NewStatusPodList() *StatusPodList {
	return &StatusPodList{
		SpecPod:  NewMyPodList(),
		OtherPod: NewMyPodList(),
	}
}

//Classify the PodList to our custom PodList
func ClassifyToMyPodList(ctx context.Context, myPodList *MyPodList, pod *corev1.Pod) error {

	if pod.DeletionTimestamp != nil {
		myPodList.DeletePod = append(myPodList.DeletePod, pod)
		myPodList.DeleteReplica++
		return nil
	}

	if pod.Status.Phase == corev1.PodRunning {
		myPodList.RunningPod = append(myPodList.RunningPod, pod)
		myPodList.RunningReplica++
	} else if pod.Status.Phase == corev1.PodPending {
		myPodList.PendingPod = append(myPodList.PendingPod, pod)
		myPodList.PendingReplica++
	} else {
		//todo  other status pod
		return errors.New("Bad pod status")
	}
	return nil
}
