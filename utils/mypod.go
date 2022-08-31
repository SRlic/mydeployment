package utils

import (
	"errors"
	"fmt"

	mydeployment "mydeployment/api/v1"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// MyPodList record pods with the same image,
// pods are divided into running pod, pending pod and delete pod
type MyPodList struct {
	Replica        int `json:"replica,omitempty"`
	RunningReplica int `json:"running_replica,omitempty"`
	PendingReplica int `json:"pedding_replica,omitempty"`
	DeleteReplica  int `json:"delete_replica,omitempty"`

	RunningPods []*corev1.Pod `json:"running_pod,omitempty"`
	PendingPods []*corev1.Pod `json:"pending_pod,omitempty"`
	DeletePods  []*corev1.Pod `json:"delete_pod,omitempty"`
}

// NewMyPodList init a new MyPodList
func NewMyPodList() *MyPodList {
	return &MyPodList{
		RunningPods: make([]*corev1.Pod, 0),
		PendingPods: make([]*corev1.Pod, 0),
		DeletePods:  make([]*corev1.Pod, 0),
	}
}

// ToDeployPodList  MyPodList to DeployPodList
func (m *MyPodList) ToDeployPodList() *mydeployment.DeployPodList {
	deletePod := make([]*mydeployment.DeployPod, 0)
	runningPod := make([]*mydeployment.DeployPod, 0)
	pendingPod := make([]*mydeployment.DeployPod, 0)

	for i := range m.DeletePods {
		deletePod = append(deletePod, mydeployment.NewDeployPod(m.DeletePods[i]))
	}
	for i := range m.RunningPods {
		runningPod = append(runningPod, mydeployment.NewDeployPod(m.RunningPods[i]))
	}
	for i := range m.PendingPods {
		pendingPod = append(pendingPod, mydeployment.NewDeployPod(m.PendingPods[i]))
	}
	return &mydeployment.DeployPodList{
		RunningReplica: m.RunningReplica,
		RunningPod:     runningPod,
		PendingReplica: m.PendingReplica,
		PendingPod:     pendingPod,
		DeleteReplica:  m.DeleteReplica,
		DeletePod:      deletePod,
	}
}

// AppendToMyPodList divides the pod into running pod, pending pod and delete pod
// If the DeletionTimestamp in pod is not Zero, the pod is deleted, append it into DeletePods
// If the status phase in pod is pending append it into PendingPods
// If the status phase in pod is running append it into RunningPods
func (m *MyPodList) AppendToMyPodList(pod *corev1.Pod) error {
	if !pod.DeletionTimestamp.IsZero() {
		m.DeletePods = append(m.DeletePods, pod)
		m.DeleteReplica++
		return nil
	}
	m.Replica++

	switch pod.Status.Phase {
	case corev1.PodRunning:
		m.RunningPods = append(m.RunningPods, pod)
		m.RunningReplica++
		return nil
	case corev1.PodPending:
		m.PendingPods = append(m.PendingPods, pod)
		m.PendingReplica++
		return nil
	default:
		return errors.New("bad pod status")
	}
}

// StatusPodList record the pod owned by current deployment
// We process pod scaling or updating based on StatusPodList
type StatusPodList struct {
	CurrentSpecReplica int        `json:"current_spec_replica,omitempty"`
	SpecPodList        *MyPodList `json:"spec_pod,omitempty"`

	// current other replica= OtherPod.RunningReplica + OtherPod.PendingReplica
	CurrentOtherReplica int        `json:"current_other_replica,omitempty"`
	OtherPodList        *MyPodList `json:"other_pod,omitempty"`
}

// ToDeploymentStatus  StatusPodList to DeploymentStatus
func (s *StatusPodList) ToDeploymentStatus() *mydeployment.MyDeploymentStatus {
	return &mydeployment.MyDeploymentStatus{
		CurrentReplica:      s.CurrentSpecReplica + s.CurrentOtherReplica,
		CurrentSpecReplica:  s.CurrentSpecReplica,
		CurrentOtherReplica: s.CurrentOtherReplica,
		SpecPod:             s.SpecPodList.ToDeployPodList(),
		OtherPod:            s.OtherPodList.ToDeployPodList(),
	}
}

// NewStatusPodList initialize a new StatusPodList
// StatusPodList record the pods in current deployment.
// The pods are divided into SpecPod and OtherPod，
func NewStatusPodList(podList *corev1.PodList, specImage string) (*StatusPodList, error) {
	logr := log.Log.WithName("myDeployment")
	specPodList := NewMyPodList()
	otherPodList := NewMyPodList()

	for i := range podList.Items {
		switch image := GetImageStrFromPod(&podList.Items[i]); image {
		case specImage:
			err := specPodList.AppendToMyPodList(&podList.Items[i])
			if err != nil {
				logr.Error(err, fmt.Sprintf("Init myDeployment error at ClassifyToPodList pod :%v", podList.Items[i]))
				return nil, err
			}
		default:
			err := otherPodList.AppendToMyPodList(&podList.Items[i])
			if err != nil {
				logr.Error(err, fmt.Sprintf("Init myDeployment error at ClassifyToPodList pod :%v", podList.Items[i]))
				return nil, err
			}
		}
	}

	return &StatusPodList{
		SpecPodList:         specPodList,
		CurrentSpecReplica:  specPodList.Replica,
		OtherPodList:        otherPodList,
		CurrentOtherReplica: otherPodList.Replica,
	}, nil
}
