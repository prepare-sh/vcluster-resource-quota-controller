package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const configMapName = "vcluster-resource-quota-controller-config"
const configMapNamespace = "default"

var clientset *kubernetes.Clientset

type Config struct {
	LimitCPU    string `json:"limitCPU"`
	LimitMemory string `json:"limitMemory"`
}

func main() {
	// Initialize the Kubernetes client
	var err error
	clientset, err = initKubernetesClient()
	if err != nil {
		log.Fatalf("Error initializing Kubernetes client: %v", err)
	}

	http.HandleFunc("/validate", handleAdmission)
	log.Println("Starting server on :8443...")
	log.Fatal(http.ListenAndServeTLS(":8443", "/etc/webhook/certs/tls.crt", "/etc/webhook/certs/tls.key", nil))
}

func initKubernetesClient() (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error

	// Try in-cluster config
	config, err = rest.InClusterConfig()
	if err != nil {
		// If not in-cluster, try the local kubeconfig file
		kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	return kubernetes.NewForConfig(config)
}

func loadConfig() (Config, error) {
	cm, err := clientset.CoreV1().ConfigMaps(configMapNamespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
	if err != nil {
		return Config{}, err
	}

	config := Config{
		LimitCPU:    cm.Data["limitCPU"],
		LimitMemory: cm.Data["limitMemory"],
	}

	return config, nil
}

func handleAdmission(w http.ResponseWriter, r *http.Request) {
	var admissionReviewRequest admissionv1.AdmissionReview
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "could not read request body", http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(body, &admissionReviewRequest); err != nil {
		http.Error(w, "could not unmarshal request", http.StatusBadRequest)
		return
	}

	admissionResponse := processAdmissionReview(admissionReviewRequest)
	admissionReviewResponse := admissionv1.AdmissionReview{
		TypeMeta: admissionReviewRequest.TypeMeta,
		Response: admissionResponse,
	}

	respBytes, err := json.Marshal(admissionReviewResponse)
	if err != nil {
		http.Error(w, "could not marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
}

func processAdmissionReview(ar admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	podResource := "pods"
	if ar.Request.Resource.Resource != podResource {
		return &admissionv1.AdmissionResponse{Allowed: true}
	}

	var pod corev1.Pod
	if err := json.Unmarshal(ar.Request.Object.Raw, &pod); err != nil {
		return &admissionv1.AdmissionResponse{Result: &metav1.Status{Message: "could not unmarshal pod object"}, Allowed: false}
	}

	config, err := loadConfig()
	if err != nil {
		return &admissionv1.AdmissionResponse{Result: &metav1.Status{Message: fmt.Sprintf("could not load config: %v", err)}, Allowed: false}
	}

	admissionResponse := &admissionv1.AdmissionResponse{UID: ar.Request.UID}

	if managedBy, ok := pod.Labels["vcluster.loft.sh/managed-by"]; ok {
		totalCPUUsage, totalMemoryUsage, err := calculateResourceUsage(ar.Request.Namespace, managedBy)
		if err != nil {
			admissionResponse.Result = &metav1.Status{Message: fmt.Sprintf("could not list pods: %v", err)}
			admissionResponse.Allowed = false
			return admissionResponse
		}

		cpuLimit := resource.MustParse(config.LimitCPU)
		memLimit := resource.MustParse(config.LimitMemory)

		for _, container := range pod.Spec.Containers {
			if err := validateResource(container.Resources, totalCPUUsage, totalMemoryUsage, cpuLimit, memLimit); err != nil {
				admissionResponse.Result = &metav1.Status{Message: err.Error()}
				admissionResponse.Allowed = false
				return admissionResponse
			}
		}
	}

	admissionResponse.Allowed = true
	return admissionResponse
}

func calculateResourceUsage(namespace, managedBy string) (resource.Quantity, resource.Quantity, error) {
	pods, err := getPodsWithLabel(namespace, "vcluster.loft.sh/managed-by", managedBy)
	if err != nil {
		return resource.Quantity{}, resource.Quantity{}, err
	}

	totalCPUUsage := resource.Quantity{}
	totalMemoryUsage := resource.Quantity{}

	for _, p := range pods {
		for _, container := range p.Spec.Containers {
			if container.Resources.Limits != nil {
				totalCPUUsage.Add(container.Resources.Limits[corev1.ResourceCPU])
				totalMemoryUsage.Add(container.Resources.Limits[corev1.ResourceMemory])
			}
		}
	}

	return totalCPUUsage, totalMemoryUsage, nil
}

func validateResource(resources corev1.ResourceRequirements, totalCPUUsage, totalMemoryUsage, cpuLimit, memLimit resource.Quantity) error {
	if resources.Limits == nil || resources.Requests == nil {
		return fmt.Errorf("container must specify both resource limits and requests")
	}

	totalCPUUsage.Add(resources.Limits[corev1.ResourceCPU])
	totalMemoryUsage.Add(resources.Limits[corev1.ResourceMemory])

	if totalCPUUsage.Cmp(cpuLimit) > 0 {
		return fmt.Errorf("CPU limit exceeded")
	}

	if totalMemoryUsage.Cmp(memLimit) > 0 {
		return fmt.Errorf("Memory limit exceeded")
	}

	return nil
}

func getPodsWithLabel(namespace, key, value string) ([]corev1.Pod, error) {
	podList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", key, value),
	})
	if err != nil {
		return nil, err
	}
	return podList.Items, nil
}
