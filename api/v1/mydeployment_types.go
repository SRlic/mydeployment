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

package v1

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DeploymentScaling   string = "Scaling"
	DeploymentRuning    string = "Runing"
	DeploymentUpgrating string = "Upgrading"
)

// SimplePod records simple information of the pod
type SimplePod struct {
	Name              string          `json:"name,omitempty"`
	Image             string          `json:"image,omitempty"`
	DeletionTimestamp *metav1.Time    `json:"deletiontimestamp,omitempty"`
	Phase             corev1.PodPhase `json:"phase,omitempty"`
}

func (dp SimplePod) String() string {
	return fmt.Sprintf("Name %v,Image %v, DeletionTimestamp %v, Phase %v ", dp.Name, dp.Image, dp.DeletionTimestamp, dp.Phase)
}

// NewSimplePod initialize a new SimplePod
func NewSimplePod(pod *corev1.Pod) *SimplePod {
	// There should be only one container in this pod
	if pod == nil || len(pod.Spec.Containers) != 1 {
		return &SimplePod{}
	}
	return &SimplePod{
		Name:              pod.Name,
		Image:             pod.Spec.Containers[0].Image,
		DeletionTimestamp: pod.DeletionTimestamp,
		Phase:             pod.Status.Phase,
	}
}

// MyDeploymentSpec defines the desired state of MyDeployment
type MyDeploymentSpec struct {
	//Replica of pod
	Replica int `json:"replica,omitempty"`
	//Container Image of pod, the num of container in one pod is limited to one in our design
	Image string `json:"image,omitempty"`
}

// MyDeploymentStatus defines the observed state of MyDeployment
type MyDeploymentStatus struct {
	AlivePodNum int    `json:"alive_pod_num,omitempty"`
	Phase       string `json:"phase,omitempty"`

	PodList []*SimplePod `json:"pod_list,omitempty"`
}

func (s MyDeploymentStatus) String() string {
	return fmt.Sprintf("alive pod num: %v, phase: %v, podList: %v", s.AlivePodNum, s.Phase, s.PodList)
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MyDeployment is the Schema for the mydeployments API
type MyDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MyDeploymentSpec   `json:"spec,omitempty"`
	Status MyDeploymentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MyDeploymentList contains a list of MyDeployment
type MyDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MyDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MyDeployment{}, &MyDeploymentList{})
}
