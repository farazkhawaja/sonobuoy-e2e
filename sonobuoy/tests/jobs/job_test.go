package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	"testing"
)

var clientset *kubernetes.Clientset

var _ = BeforeSuite(func() {
	var config *rest.Config
	var err error

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

// Job CRUD test suite
var _ = Describe("Jobs CRUD Operations", func() {
	var namespace string
	var jobName string

	BeforeEach(func() {
		namespace = os.Getenv("TEST_NAMESPACE")
		if namespace == "" {
			namespace = "default"
		}
		jobName = fmt.Sprintf("test-job-%d", time.Now().UnixNano())

		// Create a Job before each test
		job := &v1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jobName,
				Namespace: namespace,
			},
			Spec: v1.JobSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:    "basic-task",
								Image:   "alpine",
								Command: []string{"sh", "-c", "echo 'Calculating something basic'"},
							},
						},
						RestartPolicy: corev1.RestartPolicyNever,
					},
				},
			},
		}

		_, err := clientset.BatchV1().Jobs(namespace).Create(context.TODO(), job, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to create job")
	})

	// Read the Job
	It("should read the job successfully", func() {
		job, err := clientset.BatchV1().Jobs(namespace).Get(context.TODO(), jobName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to read job")
		Expect(job.Name).To(Equal(jobName))
	})

	//// Update the Job
	//It("should update the job successfully", func() {
	//	// Get the job and modify it
	//	job, err := clientset.BatchV1().Jobs(namespace).Get(context.TODO(), jobName, metav1.GetOptions{})
	//	Expect(err).NotTo(HaveOccurred(), "Failed to get job for update")
	//
	//	job.Spec.Template.Spec.Containers[0].Command = []string{"perl", "-Mbignum=bpi", "-wle", "print bpi(1000)"}
	//	_, err = clientset.BatchV1().Jobs(namespace).Update(context.TODO(), job, metav1.UpdateOptions{})
	//	Expect(err).NotTo(HaveOccurred(), "Failed to update job")
	//})

	// Delete the Job
	AfterEach(func() {
		// Ensure the Job exists before trying to delete it
		_, err := clientset.BatchV1().Jobs(namespace).Get(context.TODO(), jobName, metav1.GetOptions{})
		if err == nil { // Only delete if it exists
			propagationPolicy := metav1.DeletePropagationOrphan
			err = clientset.BatchV1().Jobs(namespace).Delete(context.TODO(), jobName, metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
			Expect(err).NotTo(HaveOccurred(), "Failed to delete job")
		}
	})
})

// Entry point for running the Ginkgo tests
func TestJobsCRUD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Jobs Test Suite")
}
