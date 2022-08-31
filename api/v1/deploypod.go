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
