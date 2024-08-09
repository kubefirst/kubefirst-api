/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k8s

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type PortForwardAPodRequest struct {
	// RestConfig is the kubernetes config
	RestConfig *rest.Config
	// Pod is the selected pod for this port forwarding
	Pod v1.Pod
	// LocalPort is the local port that will be selected to expose the PodPort
	LocalPort int
	// PodPort is the target port for the pod
	PodPort int

	//// Steams configures where to write or read input from
	// Streams genericclioptions.IOStreams

	// StopCh is the channel used to manage the port forward lifecycle
	StopCh <-chan struct{}
	// ReadyCh communicates when the tunnel is ready to receive traffic
	ReadyCh chan struct{}
}

type PortForwardAServiceRequest struct {
	// RestConfig is the kubernetes config
	RestConfig *rest.Config
	// Service is the selected service for this port forwarding
	Service v1.Service
	// LocalPort is the local port that will be selected to expose the ServicePort
	LocalPort int
	// ServicePort is the target port for the service
	ServicePort int

	//// Steams configures where to write or read input from
	// Streams genericclioptions.IOStreams

	// StopCh is the channel used to manage the port forward lifecycle
	StopCh <-chan struct{}
	// ReadyCh communicates when the tunnel is ready to receive traffic
	ReadyCh chan struct{}
}

func PortForwardPodWithRetry(clientset *kubernetes.Clientset, req PortForwardAPodRequest) error {
	var err error
	for i := 0; i < 10; i++ {
		err = PortForwardPod(clientset, req)
		if err == nil {
			return nil
		}
		time.Sleep(20 * time.Second)
	}

	return fmt.Errorf("not able to open port-forward: %w", err)
}

// PortForwardPod receives a PortForwardAPodRequest, and enables port forwarding for the specified resource.
// If the provided Pod name matches a running Pod, it will try to port forward for that Pod on the specified port.
func PortForwardPod(clientset *kubernetes.Clientset, req PortForwardAPodRequest) error {
	podList, err := clientset.CoreV1().Pods(req.Pod.Namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil || len(podList.Items) == 0 {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var runningPod *v1.Pod
	for _, pod := range podList.Items {
		// pick the first pod found to be running
		if pod.Status.Phase == v1.PodRunning && strings.HasPrefix(pod.Name, req.Pod.Name) {
			runningPod = pod.DeepCopy()
			break
		}
	}
	if runningPod == nil {
		return fmt.Errorf("error reading pod details")
	}

	log.Println("Namespace for PF", runningPod.Namespace)
	log.Println("Name for PF", runningPod.Name)

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", runningPod.Namespace, runningPod.Name)
	hostURL, err := url.Parse(req.RestConfig.Host)
	if err != nil {
		return fmt.Errorf("could not parse kubernetes host url: %w", err)
	}

	if hostURL.Host == "" {
		hostURL.Host = req.RestConfig.Host
	}
	transport, upgrader, err := spdy.RoundTripperFor(req.RestConfig)
	if err != nil {
		return err
	}

	dialer := spdy.NewDialer(
		upgrader,
		&http.Client{Transport: transport},
		http.MethodPost,
		&url.URL{
			Scheme: "https",
			Path:   path,
			Host:   hostURL.Host,
		},
	)

	fw, err := portforward.New(
		dialer,
		[]string{fmt.Sprintf(
			"%d:%d",
			req.LocalPort,
			req.PodPort)},
		req.StopCh,
		req.ReadyCh,
		nil,
		nil)
	if err != nil {
		return err
	}

	err = fw.ForwardPorts()
	if err != nil {
		return err
	}

	return nil
}
