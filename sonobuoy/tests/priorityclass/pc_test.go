package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
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

var _ = Describe("PriorityClass CRUD Operations", func() {
	var priorityClassName string

	BeforeEach(func() {
		priorityClassName = fmt.Sprintf("test-priorityclass-%d", time.Now().UnixNano())

		// Create a PriorityClass before each test
		priorityClass := &v1.PriorityClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: priorityClassName,
			},
			Value:         1000,
			GlobalDefault: false,
			Description:   "Test Priority Class",
		}

		_, err := clientset.SchedulingV1().PriorityClasses().Create(context.TODO(), priorityClass, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to create PriorityClass")
	})

	It("should read the PriorityClass successfully", func() {
		priorityClass, err := clientset.SchedulingV1().PriorityClasses().Get(context.TODO(), priorityClassName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to read PriorityClass")
		Expect(priorityClass.Value).To(Equal(int32(1000)))
	})

	AfterEach(func() {
		// Delete the PriorityClass after each test
		err := clientset.SchedulingV1().PriorityClasses().Delete(context.TODO(), priorityClassName, metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to delete PriorityClass")
	})
})

// Entry point for running the Ginkgo tests
func TestPriorityClassCRUD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PriorityClass Test Suite")
}
