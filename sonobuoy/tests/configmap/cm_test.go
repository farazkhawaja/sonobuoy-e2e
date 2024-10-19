package e2e

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

// ConfigMap CRUD test suite with unique configmap names
var _ = Describe("ConfigMap CRUD Operations", func() {
	var namespace string
	var configMapName string

	BeforeEach(func() {
		// Define namespace and generate a unique ConfigMap name with a timestamp
		namespace = os.Getenv("TEST_NAMESPACE")
		if namespace == "" {
			namespace = "default"
		}
		configMapName = fmt.Sprintf("test-configmap-%d", time.Now().UnixNano())

		// Create a ConfigMap before each test
		configMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: namespace,
			},
			Data: map[string]string{
				"config-key": "config-value",
			},
		}

		_, err := clientset.CoreV1().ConfigMaps(namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to create ConfigMap")
	})

	// Read the ConfigMap
	It("should read the ConfigMap successfully", func() {
		configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to read ConfigMap")
		Expect(configMap.Data["config-key"]).To(Equal("config-value"))
	})

	// Update the ConfigMap
	It("should update the ConfigMap successfully", func() {
		configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to get ConfigMap for update")

		// Modify the ConfigMap data
		configMap.Data["config-key"] = "updated-value"
		_, err = clientset.CoreV1().ConfigMaps(namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to update ConfigMap")
	})

	AfterEach(func() {
		// Ensure the ConfigMap exists before trying to delete it
		_, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
		if err == nil { // Only delete if it exists
			err = clientset.CoreV1().ConfigMaps(namespace).Delete(context.TODO(), configMapName, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to delete ConfigMap")
		}
	})
})

// Entry point for running the Ginkgo tests
func TestConfigMapCRUD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ConfigMap CRUD Suite")
}
