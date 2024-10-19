package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"
)

var clientset *kubernetes.Clientset

// Setup Kubernetes client before the tests
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

// Deployment CRUD test suite with unique deployment names
var _ = Describe("Deployment CRUD Operations", func() {
	var namespace string
	var deploymentName string

	BeforeEach(func() {
		// Define namespace and generate a unique Deployment name with a timestamp
		namespace = os.Getenv("TEST_NAMESPACE")
		if namespace == "" {
			namespace = "default"
		}
		deploymentName = fmt.Sprintf("test-deployment-%d", time.Now().UnixNano())

		// Create a Deployment before each test
		replicas := int32(1)
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deploymentName,
				Namespace: namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "test-app",
					},
				},
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "test-app",
						},
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name:    "alpine",
								Image:   "alpine",
								Command: []string{"sh", "-c", "sleep 3600"},
							},
						},
					},
				},
			},
		}

		_, err := clientset.AppsV1().Deployments(namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to create deployment")

		// Wait for the Deployment to be available
		Eventually(func() bool {
			dep, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to get deployment status")
			return dep.Status.AvailableReplicas == 1
		}, 120*time.Second, 2*time.Second).Should(BeTrue(), "Deployment was not ready within the timeout")
	})

	// Read the Deployment
	It("should read the Deployment successfully", func() {
		deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to read deployment")
		Expect(deployment.Spec.Replicas).To(Equal(int32Ptr(1)))
	})

	// Update the Deployment with Conflict Handling
	It("should update the Deployment successfully", func() {
		// Retry loop to handle conflicts
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			// Fetch the latest version of the Deployment
			deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			// Modify the Deployment spec (e.g., change the number of replicas)
			replicas := int32(2)
			deployment.Spec.Replicas = &replicas

			// Update the Deployment
			_, err = clientset.AppsV1().Deployments(namespace).Update(context.TODO(), deployment, metav1.UpdateOptions{})
			return err
		})
		Expect(err).NotTo(HaveOccurred(), "Failed to update deployment")

		// Wait for the Deployment to scale up
		Eventually(func() bool {
			dep, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to get deployment status")
			return dep.Status.AvailableReplicas == 2
		}, 120*time.Second, 2*time.Second).Should(BeTrue(), "Deployment did not scale within the timeout")
	})

	// Delete the Deployment
	AfterEach(func() {
		// Ensure the Deployment exists before trying to delete it
		_, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
		if err == nil { // Only delete if it exists
			err = clientset.AppsV1().Deployments(namespace).Delete(context.TODO(), deploymentName, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to delete deployment")
		}
	})
})

// Helper function to return a pointer to int32
func int32Ptr(i int32) *int32 {
	return &i
}

// Entry point for running the Ginkgo tests
func TestDeploymentCRUD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deployment CRUD Suite")
}
