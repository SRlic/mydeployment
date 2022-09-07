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
	AlivePodNum   int `json:"replica,omitempty"`
	RunningPodNum int `json:"running_replica,omitempty"`
	PendingPodNum int `json:"pedding_replica,omitempty"`
	DeletedPodNum int `json:"delete_replica,omitempty"`

	RunningPods []*corev1.Pod `json:"running_pod,omitempty"`
	PendingPods []*corev1.Pod `json:"pending_pod,omitempty"`
	DeletedPods []*corev1.Pod `json:"delete_pod,omitempty"`
}

// NewMyPodList init a new MyPodList
func NewMyPodList() *MyPodList {
	return &MyPodList{
		RunningPods: make([]*corev1.Pod, 0),
		PendingPods: make([]*corev1.Pod, 0),
		DeletedPods: make([]*corev1.Pod, 0),
	}
}

// ToDeployPodList  MyPodList to DeployPodList
func (m *MyPodList) ToDeployPodList() *mydeployment.DeployPodList {
	deletedPod := make([]*mydeployment.DeployPod, 0)
	runningPod := make([]*mydeployment.DeployPod, 0)
	pendingPod := make([]*mydeployment.DeployPod, 0)

	for i := range m.DeletedPods {
		deletedPod = append(deletedPod, mydeployment.NewDeployPod(m.DeletedPods[i]))
	}
	for i := range m.RunningPods {
		runningPod = append(runningPod, mydeployment.NewDeployPod(m.RunningPods[i]))
	}
	for i := range m.PendingPods {
		pendingPod = append(pendingPod, mydeployment.NewDeployPod(m.PendingPods[i]))
	}
	return &mydeployment.DeployPodList{
		RunningPodNum:  m.RunningPodNum,
		RunningPodList: runningPod,
		PendingPodNum:  m.PendingPodNum,
		PendingPodList: pendingPod,
		DeletedPodNum:  m.DeletedPodNum,
		DeletedPodList: deletedPod,
	}
}

// AppendToMyPodList divides the pod into running pod, pending pod and delete pod
// If the DeletionTimestamp in pod is not Zero, the pod is deleted, append it into DeletePods
// If the status phase in pod is pending append it into PendingPods
// If the status phase in pod is running append it into RunningPods
func (m *MyPodList) AppendToMyPodList(pod *corev1.Pod) error {
	if !pod.DeletionTimestamp.IsZero() {
		m.DeletedPods = append(m.DeletedPods, pod)
		m.DeletedPodNum++
		return nil
	}
	m.AlivePodNum++

	switch pod.Status.Phase {
	case corev1.PodRunning:
		m.RunningPods = append(m.RunningPods, pod)
		m.RunningPodNum++
		return nil
	case corev1.PodPending:
		m.PendingPods = append(m.PendingPods, pod)
		m.PendingPodNum++
		return nil
	default:
		return errors.New("bad pod status")
	}
}

// StatusPodList record the pod owned by current deployment
// We process pod scaling or updating based on StatusPodList
type StatusPodList struct {
	AlivePodNum int `json:"alive_pod_num,omitempty"`

	AliveSpecPodNum int        `json:"alive_spec_pod_num,omitempty"`
	SpecPodList     *MyPodList `json:"spec_pod,omitempty"`

	// current other replica= OtherPod.RunningPodNum + OtherPod.PendingPodNum
	AliveExpiredPodNum int        `json:"current_other_replica,omitempty"`
	ExpiredPodList     *MyPodList `json:"other_pod,omitempty"`
}

// ToDeploymentStatus  StatusPodList to DeploymentStatus
func (s *StatusPodList) ToDeploymentStatus() *mydeployment.MyDeploymentStatus {
	return &mydeployment.MyDeploymentStatus{
		AlivePodNum:        s.AlivePodNum,
		AliveSpecPodNum:    s.AliveSpecPodNum,
		AliveExpiredPodNum: s.AliveExpiredPodNum,
		SpecPodList:        s.SpecPodList.ToDeployPodList(),
		ExpiredPodList:     s.ExpiredPodList.ToDeployPodList(),
	}
}

func (s *StatusPodList) NeedScale(specReplica int) bool {
	if s.AlivePodNum == specReplica {
		return false
	}
	return true
}

func (s *StatusPodList) NeedUpgrade() bool {
	if s.AliveExpiredPodNum > 0 {
		return true
	}
	return false
}

func (s *StatusPodList) SpecPodPending() bool {
	if s.SpecPodList.PendingPodNum > 0 {
		return true
	}
	return false
}

func (s *StatusPodList) ExpiredPodPending() bool {
	if s.ExpiredPodList.PendingPodNum > 0 {
		return true
	}
	return false
}

// NewStatusPodList initialize a new StatusPodList
// StatusPodList record the pods in current deployment.
// The pods are divided into SpecPod and OtherPod，
func NewStatusPodList(podList *corev1.PodList, specImage string) (*StatusPodList, error) {
	logr := log.Log.WithName("myDeployment")
	specPodList := NewMyPodList()
	expiredPodList := NewMyPodList()

	for i := range podList.Items {
		switch image := GetImageStrFromPod(&podList.Items[i]); image {
		case specImage:
			err := specPodList.AppendToMyPodList(&podList.Items[i])
			if err != nil {
				logr.Error(err, fmt.Sprintf("Init myDeployment error at ClassifyToPodList pod :%v", podList.Items[i]))
				return nil, err
			}
		default:
			err := expiredPodList.AppendToMyPodList(&podList.Items[i])
			if err != nil {
				logr.Error(err, fmt.Sprintf("Init myDeployment error at ClassifyToPodList pod :%v", podList.Items[i]))
				return nil, err
			}
		}
	}

	return &StatusPodList{
		AlivePodNum:        specPodList.AlivePodNum + expiredPodList.AlivePodNum,
		SpecPodList:        specPodList,
		AliveSpecPodNum:    specPodList.AlivePodNum,
		ExpiredPodList:     expiredPodList,
		AliveExpiredPodNum: expiredPodList.AlivePodNum,
	}, nil
}
