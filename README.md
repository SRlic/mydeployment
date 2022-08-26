# mydeployment
A simple Implement of deployment base on Kubebuilder which support pod deployment，scale up, scale down, and rolling update based on image and replica in your mydeployment cr

## Description
A simple Implement of deployment. In our design, Deployment directly operates pods. The main functions include pod deployment, expansion, shrinkage and rolling update. The granularity of rolling update is 1.

MyDeployment Resource:
```yaml
tapiVersion: kubelearn.liyichen.kubebuilder.io/v1
kind: MyDeployment
metadata:
  name: mydeployment-sample
spec:
   replica: 4
   image: nginx
  # TODO(user): Add fields here

```
The MyDeployment status includes running，scaling and updating。

Points to note:
1. When we scale up or update our deployment，the deployment may be blocked by some pending pods (the pods we want to pull up), our deployment will not  actively delete pending pods。If your deployment is keeping on scaling or updating，you can manually delete the pending pod，you can also modify the replica  or image to trigger the deletion of the pod. Pod deletion will delete pending pod firstly
2. Only support single image 
3. We default that the pod will be deleted successfully

Todo list:

1. Support multiple images
2. More detailed MyDeployment status
3. ...

## Getting Started
You’ll need a Kubernetes cluster before starting，see more details about k8s at [kubernetes](https://kubernetes.io/).
This project is write based on kubebuilder, see more details about kubebuilder at [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder)
Install the CRDs into the cluster:
```sh
make install
```
Run controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
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
Yichen Li 

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

