package utils

import (
	"context"
	"math/rand"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	BATCHUPDATESIZE = 1  //  Rolling update granularity
	PODNAMESIZE     = 10 // Suffix length of a pod name
)

const (
	letters = "abcdefghijklmnopqrstuvwxyz0123456789"
)

// RandStr generate random string for pod name
func RandStr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// GetImageStrFromPod get the image of pod
func GetImageStrFromPod(pod *corev1.Pod) string {
	var res string = ""
	if pod == nil {
		return ""
	}
	// Only one container in this pod, we only suppose one container per pod this version
	if len(pod.Spec.Containers) == 1 {
		return pod.Spec.Containers[0].Image
	}
	// Multi container is not support for our deployment
	// ";;" should be illegal in image name, we use ";;" as the special delimiter
	for i := range pod.Spec.Containers {
		res = res + pod.Spec.Containers[i].Image + ";;"
	}
	return res
}

// ImageStrToArr split image string to image string array
func ImageStrToArr(image string) []string {
	vec := strings.Split(image, ";;")
	return vec
}

// UpdateWithRetry use RetryOnConflict make an update to a resource when other code making unrelated updates to the resource at the same time
func UpdateWithRetry(ctx context.Context, kClient client.Client,
	object client.Object, mutate func()) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := kClient.Get(ctx, client.ObjectKeyFromObject(object), object)
		if err != nil {
			return err
		}

		mutate()

		return kClient.Update(ctx, object)
	})
}
