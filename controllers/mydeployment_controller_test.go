package controllers

import (
	"context"
	"sync"
	"time"

	mydeployment "mydeployment/api/v1"
	"mydeployment/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("BackupExecution Controller", func() {
	var (
		ctx = context.TODO()
	)

	Context("when create mydeployment", func() {
		var (
			managerCtx  context.Context
			managerStop func()
			wg          sync.WaitGroup
		)
		BeforeEach(func() {
			managerCtx, managerStop = context.WithCancel(context.Background())
			k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
				Scheme:             scheme.Scheme,
				MetricsBindAddress: "0",
			})
			Expect(err).NotTo(HaveOccurred())

			err = (&MyDeploymentReconciler{
				Client: k8sManager.GetClient(),
				Scheme: k8sManager.GetScheme(),
			}).SetupWithManager(k8sManager)
			Expect(err).NotTo(HaveOccurred())

			wg = sync.WaitGroup{}
			wg.Add(1)

			go func() {
				defer GinkgoRecover()
				err := k8sManager.Start(managerCtx)
				Expect(err).ToNot(HaveOccurred(), "failed to run manager")
				wg.Done()
			}()
		})

		AfterEach(func() {
			Eventually(func() error {
				err := k8sClient.DeleteAllOf(ctx, &mydeployment.MyDeployment{}, client.InNamespace(testNamespace))
				if err != nil && !apierrors.IsNotFound(err) {
					return err
				}

				list := &mydeployment.MyDeploymentList{}

				err = k8sClient.List(ctx, list, client.InNamespace(testNamespace))
				if err != nil {
					return err
				}

				if len(list.Items) != 0 {
					return errors.Errorf("expected deploymentList to be empty, but got length %d", len(list.Items))
				}

				return nil
			}, timeout, interval).ShouldNot(HaveOccurred())

			managerStop()
			wg.Wait()
		})

		It("Should create pods described by image and replica in deployment", func() {
			const (
				deploymentName = "mydeployment-test1"
				ImageName      = "tomcat"
				SpecReplica    = 1
			)
			// var (
			// 	deploymentName = []string{"mydeployment-test1", "mydeployment-test2"}
			// )
			By("By create a new deployment")
			deployment := &mydeployment.MyDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      deploymentName,
				},
				Spec: mydeployment.MyDeploymentSpec{
					Image:   ImageName,
					Replica: SpecReplica,
				},
			}
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
			By("By checking pod has been created ")
			checkPodCreatedByDeployment(ctx, deployment)
		})
		It("Should scale pods", func() {
			const (
				deploymentName = "mydeployment-test1"
				ImageName      = "tomcat"
				SpecReplica    = 1
			)

			By("By create a new deployment")
			deployment := &mydeployment.MyDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      deploymentName,
				},
				Spec: mydeployment.MyDeploymentSpec{
					Image:   ImageName,
					Replica: SpecReplica,
				},
			}
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
			By("By checking pods are created ")
			checkPodCreatedByDeployment(ctx, deployment)

			By("By checking the pods scale up")
			time.Sleep(5 * time.Second)
			UpdateDeploymentSpec(ctx, deployment, func() {
				deployment.Spec.Replica = 5
			})
			checkPodCreatedByDeployment(ctx, deployment)

			By("By checking the pods scale down")
			time.Sleep(5 * time.Second)
			UpdateDeploymentSpec(ctx, deployment, func() {
				deployment.Spec.Replica = 3
			})
			checkPodCreatedByDeployment(ctx, deployment)
		})

		It("Should scale pods", func() {
			const (
				deploymentName = "mydeployment-test1"
				ImageName      = "tomcat"
				SpecReplica    = 5
			)

			By("By create a new deployment")
			deployment := &mydeployment.MyDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      deploymentName,
				},
				Spec: mydeployment.MyDeploymentSpec{
					Image:   ImageName,
					Replica: SpecReplica,
				},
			}
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
			By("By checking pods are created ")
			checkPodCreatedByDeployment(ctx, deployment)

			By("By checking the pods upgrade")
			time.Sleep(20 * time.Second)
			UpdateDeploymentSpec(ctx, deployment, func() {
				deployment.Spec.Image = "nginx"
			})
			checkPodCreatedByDeployment(ctx, deployment)

			By("By checking the pods upgrade")
			time.Sleep(20 * time.Second)
			UpdateDeploymentSpec(ctx, deployment, func() {
				deployment.Spec.Replica = 3
				deployment.Spec.Image = "tomcat"
			})
			checkPodCreatedByDeployment(ctx, deployment)

			By("By checking the pods upgrade")
			time.Sleep(20 * time.Second)
			UpdateDeploymentSpec(ctx, deployment, func() {
				deployment.Spec.Replica = 5
				deployment.Spec.Image = "nginx"
			})
			checkPodCreatedByDeployment(ctx, deployment)
		})
	})
})

func checkDeploymentStatus(ctx context.Context, deployment *mydeployment.MyDeployment) {
	podList := &corev1.PodList{}
	Eventually(func() bool {
		err := k8sClient.Get(ctx, types.NamespacedName{Namespace: testNamespace, Name: deployment.Name}, deployment)
		if err != nil {
			return false
		}
		err = k8sClient.List(ctx, podList, client.InNamespace(testNamespace), client.MatchingLabels{"app": deployment.Name})
		if err != nil {
			return false
		}
		for i := range podList.Items {
			if podList.Items[i].Status.Phase == corev1.PodPending {
				return deployment.Status.Phase == mydeployment.DeploymentScaling
			}
		}
		return deployment.Status.Phase == mydeployment.DeploymentRuning
	}, timeout, interval).Should(BeTrue())
}

func checkPodCreatedByDeployment(ctx context.Context, deployment *mydeployment.MyDeployment) *corev1.PodList {
	podList := &corev1.PodList{}
	By("By checking created podlist fields")
	Eventually(func() bool {
		err := k8sClient.List(ctx, podList, client.InNamespace(testNamespace), client.MatchingLabels{"app": deployment.Name})
		if err != nil {
			return false
		}
		if len(podList.Items) != deployment.Spec.Replica {
			return false
		}

		for i := range podList.Items {
			if utils.GetImageStrFromPod(&podList.Items[i]) != deployment.Spec.Image {
				return false
			}
		}
		return true
	}, timeout, interval).Should(BeTrue())
	return podList
}

func UpdateDeploymentSpec(ctx context.Context, mydeployment *mydeployment.MyDeployment, mutate func()) {
	By("update spec of deployment " + mydeployment.Name)

	err := utils.UpdateWithRetry(ctx, k8sClient, mydeployment, mutate)
	Expect(err).NotTo(HaveOccurred())
}
