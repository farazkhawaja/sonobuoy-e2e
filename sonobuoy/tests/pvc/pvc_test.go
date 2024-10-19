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

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
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

var _ = Describe("PVC and Pod Operations", func() {
	var namespace string
	var pvcName string
	var podName string

	BeforeEach(func() {
		namespace = os.Getenv("TEST_NAMESPACE")
		if namespace == "" {
			namespace = "default"
		}
		pvcName = fmt.Sprintf("test-pvc-%d", time.Now().UnixNano())
		podName = fmt.Sprintf("test-pod-pvc-%d", time.Now().UnixNano())

		// Create a PVC
		pvc := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pvcName,
				Namespace: namespace,
			},
			Spec: v1.PersistentVolumeClaimSpec{
				AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceStorage: resource.MustParse("10Mi"),
					},
				},
			},
		}

		_, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to create PVC")

		// Wait for PVC to be bound
		Eventually(func() bool {
			pvc, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), pvcName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to get PVC status")
			return pvc.Status.Phase == v1.ClaimBound
		}, 120*time.Second, 2*time.Second).Should(BeTrue(), "PVC was not bound within the timeout")
	})

	It("should create a pod and mount the PVC successfully", func() {
		// Create a pod that mounts the PVC using the lightweight Alpine image
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: namespace,
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:    "alpine-container",
						Image:   "alpine", // Lightweight image
						Command: []string{"sh", "-c", "sleep 3600"},
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      "pvc-volume",
								MountPath: "/mnt/test",
							},
						},
					},
				},
				Volumes: []v1.Volume{
					{
						Name: "pvc-volume",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
								ClaimName: pvcName,
							},
						},
					},
				},
			},
		}

		_, err := clientset.CoreV1().Pods(namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to create pod")

		// Wait for the pod to be running
		Eventually(func() bool {
			pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to get pod")
			return pod.Status.Phase == v1.PodRunning
		}, 120*time.Second, 2*time.Second).Should(BeTrue(), "Pod did not reach running state within the timeout")
	})

	AfterEach(func() {
		// Cleanup: delete the pod and PVC
		err := clientset.CoreV1().Pods(namespace).Delete(context.TODO(), podName, metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to delete pod")

		err = clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(context.TODO(), pvcName, metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred(), "Failed to delete PVC")
	})
})

func TestPVCPodOperations(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PVC and Pod Operations Suite")
}
