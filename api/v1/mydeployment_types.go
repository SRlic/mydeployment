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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DeployScaling   string = "Scaling"
	DepolyRuning    string = "Runing"
	DepolyUpgrating string = "Upgrading"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MyDeploymentSpec defines the desired state of MyDeployment
type MyDeploymentSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	//Replica of pod
	Replica int `json:"replica,omitempty"`
	//Container Image of pod, the num of container in one pod is limited to one in our design
	Image string `json:"image,omitempty"`
}

// MyDeploymentStatus defines the observed state of MyDeployment
type MyDeploymentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	//current replica= Specpod.RunningReplica + Specpod.PendingReplica + OtherPod.RunningReplica + OtherPod.PendingReplica
	AlivePodNum int    `json:"alive_pod_num,omitempty"`
	Phase       string `json:"phase,omitempty"`

	//current spec replica= Specpod.RunningReplica + Specpod.PendingReplica
	AliveSpecPodNum int            `json:"alive_spec_pod_num,omitempty"`
	SpecPodList     *DeployPodList `json:"spec_pod_list,omitempty"`

	//current other replica= OtherPod.RunningReplica + OtherPod.PendingReplica
	AliveExpiredPodNum int            `json:"alive_expired_pod_num,omitempty"`
	ExpiredPodList     *DeployPodList `json:"expired_pod_list,omitempty"`
}

// UpdateStatusPhase update the MyDeploymentStatus's phase
func (s *MyDeploymentStatus) UpdateStatusPhase(spec *MyDeploymentSpec) {
	if s.ExpiredPodList.DeletedPodNum > 0 {
		s.Phase = DepolyUpgrating
		return
	}

	if s.AliveExpiredPodNum == 0 {
		if s.AliveSpecPodNum != spec.Replica {
			s.Phase = DeployScaling
		} else {
			if s.SpecPodList.PendingPodNum > 0 || s.SpecPodList.DeletedPodNum > 0 {
				s.Phase = DeployScaling
			} else {
				s.Phase = DepolyRuning
			}
		}
	} else {
		s.Phase = DepolyUpgrating
	}
}

func (s MyDeploymentStatus) String() string {
	res := fmt.Sprintf("Spec pod: \r\n%v",
		s.SpecPodList)
	res += "\r\n"
	res += fmt.Sprintf("Other pod: \r\n%v",
		s.ExpiredPodList)
	return res
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
