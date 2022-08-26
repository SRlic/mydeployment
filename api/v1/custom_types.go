package v1

import (
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//Custom Pod records some metadata in corev1.pod
type DeployPod struct {
	Name              string          `json:"name,omitempty"`
	Image             string          `json:"image,omitempty"`
	DeletionTimestamp *metav1.Time    `json:"deletiontimestamp,omitempty"`
	Phase             corev1.PodPhase `json:"phase,omitempty"`
}

func (dp DeployPod) String() string {
	return fmt.Sprintf("Name %v,Image %v, DeletionTimestamp %v, Phase %v ", dp.Name, dp.Image, dp.DeletionTimestamp, dp.Phase)
}

func NewDeployPod(pod *corev1.Pod) *DeployPod {
	//There should be only one container in this pod
	if pod == nil || len(pod.Spec.Containers) != 1 {
		return &DeployPod{}
	}
	return &DeployPod{
		Name:              pod.Name,
		Image:             pod.Spec.Containers[0].Image,
		DeletionTimestamp: pod.DeletionTimestamp,
		Phase:             pod.Status.Phase,
	}
}

//Custom PodList divide the pods into running pod, pending pod and delete pod
type DeployPodList struct {
	RunningReplica int `json:"running_replica,omitempty"`
	PendingReplica int `json:"pedding_replica,omitempty"`
	DeleteReplica  int `json:"delete_replica,omitempty"`

	RunningPod []*DeployPod `json:"running_pod,omitempty"`

	PendingPod []*DeployPod `json:"pending_pod,omitempty"`

	DeletePod []*DeployPod `json:"delete_pod,omitempty"`
}

func (dpl DeployPodList) String() string {
	res := fmt.Sprintf("RunningReplica %v, PendingReplica %v, DeleteReplica %v \r\n", dpl.RunningReplica, dpl.PendingReplica, dpl.DeleteReplica)
	res += "Running pod: \r\n"
	for i := range dpl.RunningPod {
		res += fmt.Sprintf("{%v} \r\n", dpl.RunningPod[i])
	}
	res += "Pending pod: \r\n"
	for i := range dpl.PendingPod {
		res += fmt.Sprintf("{%v} \r\n", dpl.PendingPod[i])
	}
	res += "Delete pod: \r\n"
	for i := range dpl.DeletePod {
		res += fmt.Sprintf("{%v} \r\n", dpl.DeletePod[i])
	}
	return res
}

func (dp *DeployPodList) Add(pod *DeployPod) error {
	if pod.DeletionTimestamp != nil {
		dp.DeletePod = append(dp.DeletePod, pod)
		dp.DeleteReplica++
		return nil
	}
	if pod.Phase == corev1.PodRunning {
		dp.RunningPod = append(dp.RunningPod, pod)
		dp.RunningReplica++
		return nil
	} else if pod.Phase == corev1.PodPending {
		dp.PendingPod = append(dp.PendingPod, pod)
		dp.PendingReplica++
		return nil
	}
	return errors.New("unkown pod status")
}
