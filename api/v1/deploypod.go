package v1

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeployPod records necessary metadata of the pod owned by Deployment
type DeployPod struct {
	Name              string          `json:"name,omitempty"`
	Image             string          `json:"image,omitempty"`
	DeletionTimestamp *metav1.Time    `json:"deletiontimestamp,omitempty"`
	Phase             corev1.PodPhase `json:"phase,omitempty"`
}

func (dp DeployPod) String() string {
	return fmt.Sprintf("Name %v,Image %v, DeletionTimestamp %v, Phase %v ", dp.Name, dp.Image, dp.DeletionTimestamp, dp.Phase)
}

// NewDeployPod initialize a new DeployPod based on pod
func NewDeployPod(pod *corev1.Pod) *DeployPod {
	// There should be only one container in this pod
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

// DeployPodList records the DeployPod with the same image.
// The pods are divided into running pod, pending pod and delete pod
type DeployPodList struct {
	RunningPodNum int `json:"running_pod_num,omitempty"`
	PendingPodNum int `json:"pending_pod_num,omitempty"`
	DeletedPodNum int `json:"deleted_pod_num,omitempty"`

	RunningPodList []*DeployPod `json:"running_pod_list,omitempty"`

	PendingPodList []*DeployPod `json:"pending_pod_list,omitempty"`

	DeletedPodList []*DeployPod `json:"deleted_pod_list,omitempty"`
}

func (dpl DeployPodList) String() string {
	res := fmt.Sprintf("RunningPodNum %v, PendingPodNUm %v, DeletedPodNum %v \r\n", dpl.RunningPodNum, dpl.PendingPodNum, dpl.DeletedPodNum)
	res += "Running pod: \r\n"
	for i := range dpl.RunningPodList {
		res += fmt.Sprintf("{%v} \r\n", dpl.RunningPodList[i])
	}
	res += "Pending pod: \r\n"
	for i := range dpl.PendingPodList {
		res += fmt.Sprintf("{%v} \r\n", dpl.PendingPodList[i])
	}
	res += "Delete pod: \r\n"
	for i := range dpl.DeletedPodList {
		res += fmt.Sprintf("{%v} \r\n", dpl.DeletedPodList[i])
	}
	return res
}
