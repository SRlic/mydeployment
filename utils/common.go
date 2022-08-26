package utils

import (
	"math/rand"
	mydeployments "mydeployment/api/v1"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const (
	BATCHUPDATESIZE = 1
	PODNAMESIZE     = 10
)

const (
	DeployScaling  string = "Scaling"
	DepolyRuning   string = "Runing"
	DepolyUpdating string = "Updating"
)

const (
	letters = "abcdefghijklmnopqrstuvwxyz0123456789"
)

//Generate random string for pod name
func RandStr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func GetImageStrFormPod(pod *corev1.Pod) string {
	var res string = ""
	if pod == nil {
		return ""
	}
	//Only one container in this pod, we only suppose one container per pod this version
	if len(pod.Spec.Containers) == 1 {
		return pod.Spec.Containers[0].Image
	}
	//Multi container is not support for our deployment
	// ;; is illegal in image name
	for i := range pod.Spec.Containers {
		res = res + pod.Spec.Containers[i].Image + ";;"
	}
	return res
}

func ImageStrToVec(image string) []string {
	vec := strings.Split(image, ";;")
	return vec
}

func GetStatusPhase(depolyment *mydeployments.MyDeployment) string {
	status := depolyment.Status
	spec := depolyment.Spec
	// Some old pods are deleting
	if status.OtherPod.DeleteReplica > 0 {
		return DepolyUpdating
	}

	if status.CurrentOtherReplica == 0 {
		//No old replica，
		if status.CurrentSpecReplica != spec.Replica {
			return DeployScaling
		} else {
			if status.SpecPod.DeleteReplica > 0 || status.SpecPod.PendingReplica > 0 {
				return DeployScaling
			}

			return DepolyRuning
		}
	} else {
		//Some old replica is running，we are update our pod now
		return DepolyUpdating
	}
}
