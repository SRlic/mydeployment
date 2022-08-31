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
	DeployScaling  string = "Scaling"
	DepolyRuning   string = "Runing"
	DepolyUpdating string = "Updating"
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
	CurrentReplica int    `json:"current_replica,omitempty"`
	Phase          string `json:"phase,omitempty"`

	//current spec replica= Specpod.RunningReplica + Specpod.PendingReplica
	CurrentSpecReplica int            `json:"current_spec_replica,omitempty"`
	SpecPod            *DeployPodList `json:"spec_pod,omitempty"`

	//current other replica= OtherPod.RunningReplica + OtherPod.PendingReplica
	CurrentOtherReplica int            `json:"current_other_replica,omitempty"`
	OtherPod            *DeployPodList `json:"other_pod,omitempty"`
}

// UpdateStatusPhase update the MyDeploymentStatus's phase
func (s *MyDeploymentStatus) UpdateStatusPhase(spec *MyDeploymentSpec) {
	if s.OtherPod.DeleteReplica > 0 {
		s.Phase = DepolyUpdating
		return
	}

	if s.CurrentOtherReplica == 0 {
		if s.CurrentSpecReplica != spec.Replica {
			s.Phase = DeployScaling
		} else {
			if s.SpecPod.DeleteReplica > 0 || s.SpecPod.PendingReplica > 0 {
				s.Phase = DeployScaling
			} else {
				s.Phase = DepolyRuning
			}
		}
	} else {
		s.Phase = DepolyUpdating
	}
}

func (s MyDeploymentStatus) String() string {
	res := fmt.Sprintf("Spec pod: \r\n%v",
		s.SpecPod)
	res += "\r\n"
	res += fmt.Sprintf("Other pod: \r\n%v",
		s.OtherPod)
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
