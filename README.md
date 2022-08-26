# mydeployment
A simple Implement of deployment base on Kubebuilder which support pod deployment，scale up, scale down, and rolling update based on image and replica in your mydeployment cr

## Description
A simple Implement of deployment. In our design, Deployment directly operates pods. The main functions include pod deployment, expansion, shrinkage and rolling update. The granularity of rolling update is 1.

Deployment Kind:
```go
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
	CurrentSpecReplica int           `json:"current_spec_replica,omitempty"`
	SpecPod            DeployPodList `json:"spec_pod,omitempty"`

	//current other replica= OtherPod.RunningReplica + OtherPod.PendingReplica
	CurrentOtherReplica int           `json:"current_other_replica,omitempty"`
	OtherPod            DeployPodList `json:"other_pod,omitempty"`
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
//api/v1/custom_types.go
//Custom Pod records some metadata in corev1.pod
type DeployPod struct {
	Name              string          `json:"name,omitempty"`
	Image             string          `json:"image,omitempty"`
	DeletionTimestamp *metav1.Time    `json:"deletiontimestamp,omitempty"`
	Phase             corev1.PodPhase `json:"phase,omitempty"`
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
```

Deployment的状态包括 running，scaling and updating。

注意点：
1. 当某些pod一直pending时，Deployment会一直处于scaling 或 updating 状态，我们可以直接kubectl 删除 pending pod，或者通过修改replica值 或修改image值来删除pending pod

todolist
1. 在当前实现中忽略了删除pod后pod的状态返回，当删除pod失败时deployment的动作需要讨论
2. 后续加入支持多image 

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.


Install the CRDs into the cluster:

```sh
>make install
```
Run controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
>make run
```


### Running on the cluster
1. Install Instances of Custom Resources:

```sh
kubectl apply -f config/samples/
```

2. Build and push your image to the location specified by `IMG`:
	
```sh
make docker-build docker-push IMG=<some-registry>/mydeployment:tag
```
	
3. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/mydeployment:tag
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller to the cluster:

```sh
make undeploy
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/) 
which provides a reconcile function responsible for synchronizing resources untile the desired state is reached on the cluster 

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

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

