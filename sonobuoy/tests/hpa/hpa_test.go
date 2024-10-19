package e2e

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var clientset *kubernetes.Clientset

var _ = BeforeSuite(func() {
	var config *rest.Config
	var err error

	// Use in-cluster config if available, or default to KUBECONFIG
	config, err = rest.InClusterConfig()
	if err != nil {
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			if home := homedir.HomeDir(); home != "" {
				kubeconfig = filepath.Join(home, ".kube", "config")
			} else {
				kubeconfig = "/root/.kube/config"
			}
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		Expect(err).NotTo(HaveOccurred(), "Failed to load kubeconfig")
	}

	clientset, err = kubernetes.NewForConfig(config)
	Expect(err).NotTo(HaveOccurred(), "Failed to create Kubernetes client")
})

var _ = Describe("HPA and Deployment Tests", func() {
	var namespace string
	var deploymentName string
	var hpaName string

	BeforeEach(func() {
		// Define the namespace and names for the HPA and deployment
		namespace = os.Getenv("TEST_NAMESPACE")
		if namespace == "" {
			namespace = "default"
		}
		deploymentName = fmt.Sprintf("test-deployment-%d", time.Now().UnixNano())
		hpaName = fmt.Sprintf("test-hpa-%d", time.Now().UnixNano())

		// Create a deployment before each test
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deploymentName,
				Namespace: namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "test",
					},
				},
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "test",
						},
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name:  "nginx",
								Image: "nginx",
							},
						},
					},
				},
			},
		}

		_, err := clientset.AppsV1().Deployments(namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to create deployment")

		// Create an HPA for the deployment
		hpa := &autoscalingv1.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      hpaName,
				Namespace: namespace,
			},
			Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
					Kind:       "Deployment",
					Name:       deploymentName,
					APIVersion: "apps/v1",
				},
				MinReplicas:                    int32Ptr(1),
				MaxReplicas:                    5,
				TargetCPUUtilizationPercentage: int32Ptr(50),
			},
		}

		_, err = clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).Create(context.TODO(), hpa, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to create HPA")
	})

	It("should read an HPA", func() {
		// Test to verify HPA creation
		hpa, err := clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).Get(context.TODO(), hpaName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to get HPA")
		Expect(hpa.Spec.MaxReplicas).To(Equal(int32(5)))
	})

	It("should scale the deployment by updating HPA", func() {
		// Get the existing HPA
		hpa, err := clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).Get(context.TODO(), hpaName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to get HPA")

		// Update the MaxReplicas and TargetCPUUtilizationPercentage to simulate a scaling change
		hpa.Spec.MaxReplicas = 10
		hpa.Spec.TargetCPUUtilizationPercentage = int32Ptr(30) // Lower the CPU threshold

		_, err = clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).Update(context.TODO(), hpa, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to update HPA")

		// Verify the changes
		updatedHPA, err := clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).Get(context.TODO(), hpaName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to get updated HPA")
		Expect(updatedHPA.Spec.MaxReplicas).To(Equal(int32(10)))
		Expect(*updatedHPA.Spec.TargetCPUUtilizationPercentage).To(Equal(int32(30)))

	})

	AfterEach(func() {
		// Clean up the HPA and deployment after each test
		err := clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).Delete(context.TODO(), hpaName, metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to delete HPA")

		err = clientset.AppsV1().Deployments(namespace).Delete(context.TODO(), deploymentName, metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to delete deployment")
	})
})

// Entry point for running the Ginkgo tests
func TestHPA(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HPA and Deployment Suite")
}

func int32Ptr(i int32) *int32 {
	return &i
}
