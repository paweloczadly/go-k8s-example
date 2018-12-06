package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type patchStringValue struct {
	Op   string `json:"op"`
	Path string `json:"path"`
}

var (
	deploymentName = os.Getenv("DEPLOYMENT_NAME")
	namespace      = os.Getenv("NAMESPACE")

	ingressName = os.Getenv("INGRESS_NAME")
	dnsDomain   = os.Getenv("DNS_DOMAIN")
	ingressHost = deploymentName + dnsDomain
)

func main() {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("Cannot fetch Kubernetes config %v\n", err)
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("Cannot create Kubernetes client %v\n", err)
	}

	log.Printf("Start cleaning resources for %s\n", deploymentName)
	patchIngress(clientset, namespace, ingressHost)
	deleteDeployment(clientset, namespace, deploymentName)
	deleteService(clientset, namespace, deploymentName)
	log.Printf("Cleanup finished sucessfully")
	log.Printf("###################################################\n")
}

func patchIngress(clientset *kubernetes.Clientset, namespace string, deploymentName string) {
	// Patch rule:
	ruleIndex, err := getIngressHostIndex(clientset, namespace, ingressName, deploymentName)
	if err != nil {
		log.Printf("Cannot find host index: %v", err)
	} else {
		payload := []patchStringValue{{
			Op:   "remove",
			Path: "/spec/rules/" + strconv.Itoa(ruleIndex),
		}}
		payloadBytes, _ := json.Marshal(payload)
		clientset.Extensions().Ingresses(namespace).Patch(ingressName, types.JSONPatchType, payloadBytes)
		log.Printf("Rule %s removed from %s ingress\n", deploymentName, ingressName)
	}

	// Patch TLS:
	tlsIndex, err := getIngressTLSIndex(clientset, namespace, ingressName, deploymentName)
	if err != nil {
		log.Printf("Cannot find TLS index: %v", err)
	} else {
		payload := []patchStringValue{{
			Op:   "remove",
			Path: "/spec/tls/" + strconv.Itoa(tlsIndex),
		}}
		payloadBytes, _ := json.Marshal(payload)
		clientset.Extensions().Ingresses(namespace).Patch(ingressName, types.JSONPatchType, payloadBytes)
		log.Printf("TLS %s removed from %s ingress\n", deploymentName, ingressName)
	}
}

func getIngressTLSIndex(clientset *kubernetes.Clientset, namespace string, ingressName string, host string) (int, error) {
	ingress, err := clientset.Extensions().Ingresses(namespace).Get(ingressName, metav1.GetOptions{})
	if err != nil {
		return -1, err
	}

	for index, tls := range ingress.Spec.TLS {
		if tls.Hosts[0] == host {
			return index, nil
		}
	}

	return -1, errors.New(host + " not found")
}

func getIngressHostIndex(clientset *kubernetes.Clientset, namespace string, ingressName string, host string) (int, error) {
	ingress, err := clientset.Extensions().Ingresses(namespace).Get(ingressName, metav1.GetOptions{})
	if err != nil {
		return -1, err
	}

	for index, rule := range ingress.Spec.Rules {
		if rule.Host == host {
			return index, nil
		}
	}

	return -1, errors.New(host + " not found")
}

func deleteDeployment(clientset *kubernetes.Clientset, namespace string, deploymentName string) {
	deletePolicy := metav1.DeletePropagationForeground
	err := clientset.Extensions().Deployments(namespace).Delete(deploymentName, &metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
	if err != nil {
		log.Printf("Cannot delete deployment: %v\n", err)
	} else {
		log.Printf("Deployment %s deleted sucessfully", deploymentName)
	}
}

func deleteService(clientset *kubernetes.Clientset, namespace string, serviceName string) {
	err := clientset.CoreV1().Services(namespace).Delete(serviceName, &metav1.DeleteOptions{})
	if err != nil {
		log.Printf("Cannot delete service: %v\n", err)
	} else {
		log.Printf("Service %s deleted sucessfully", serviceName)
	}
}
