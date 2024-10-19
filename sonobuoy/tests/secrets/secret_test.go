package e2e

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

// Secret CRUD test suite with unique secret names
var _ = Describe("Secrets CRUD Operations", func() {
	var namespace string
	var secretName string

	BeforeEach(func() {
		// Define namespace and generate a unique secret name with a timestamp
		namespace = os.Getenv("TEST_NAMESPACE")
		if namespace == "" {
			namespace = "default"
		}
		secretName = fmt.Sprintf("test-secret-%d", time.Now().UnixNano())

		// Create a secret before each test
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("secret"),
			},
			Type: v1.SecretTypeOpaque,
		}

		_, err := clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to create secret")
	})

	// Read the secret
	It("should read the secret successfully", func() {
		secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to read secret")
		Expect(secret.Data["username"]).To(Equal([]byte("admin")))
		Expect(secret.Data["password"]).To(Equal([]byte("secret")))
	})

	// Update the secret
	It("should update the secret successfully", func() {
		secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to get secret for update")

		// Modify the secret data
		secret.Data["password"] = []byte("newsecret")
		_, err = clientset.CoreV1().Secrets(namespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
		// Check if the error is a StatusError and extract errstatus.message
		if statusError, isStatus := err.(*errors.StatusError); isStatus {
			// Fail the test and only show the relevant error message
			Fail(fmt.Sprintf("Error: %s", statusError.ErrStatus.Message))
		} else {
			// If no error or unexpected error, ensure the test fails accordingly
			Expect(err).NotTo(HaveOccurred(), "Unexpected failure during secret update")
		}
	})

	AfterEach(func() {
		// Ensure the secret exists before trying to delete it
		_, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
		if err == nil { // Only delete if it exists
			err = clientset.CoreV1().Secrets(namespace).Delete(context.TODO(), secretName, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to delete secret")
		}
	})
})

// Entry point for running the Ginkgo tests
func TestSecretsCRUD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Secrets CRUD Suite")
}
