/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k8s

import (
	// b64 "encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/konstructio/kubefirst-api/internal/helpers"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type KubernetesClient struct {
	Clientset      kubernetes.Interface
	RestConfig     *rest.Config
	KubeConfigPath string
}

// CreateKubeConfig returns a struct KubernetesClient with references to a clientset,
// restConfig, and path to the Kubernetes config used to generate the client
func CreateKubeConfig(inCluster bool, kubeConfigPath string) (*KubernetesClient, error) {
	// inCluster is either true or false
	// If it's true, we pull Kubernetes API authentication from Pod SA
	// If it's false, we use local machine settings
	if inCluster {
		config, err := rest.InClusterConfig()
		if err != nil {
			log.Errorf("error creating kubernetes config: %s", err)
			return nil, fmt.Errorf("error creating kubernetes config: %w", err)
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			log.Errorf("error creating kubernetes client: %s", err)
			return nil, fmt.Errorf("error creating kubernetes client: %w", err)
		}

		return &KubernetesClient{
			Clientset:      clientset,
			RestConfig:     config,
			KubeConfigPath: "in-cluster",
		}, nil
	}

	// Set path to kubeconfig
	kubeconfig := returnKubeConfigPath(kubeConfigPath)
	fs := afero.NewOsFs()

	// Check to make sure kubeconfig actually exists
	// If it doesn't, go fetch it
	if helpers.FileExists(fs, kubeconfig) {
		log.Debug("kubeconfig exists, moving on.")
	}

	// Show what path was set for kubeconfig
	log.Debugf("setting kubeconfig to: %s", kubeconfig)

	// Build configuration instance from the provided config file
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Errorf("unable to locate kubeconfig file - checked path: %s", kubeconfig)
		return nil, fmt.Errorf("unable to locate kubeconfig file - checked path: %s", kubeconfig)
	}

	// Create clientset, which is used to run operations against the API
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorf("error creating kubernetes client: %s", err)
		return nil, fmt.Errorf("error creating kubernetes client: %w", err)
	}

	return &KubernetesClient{
		Clientset:      clientset,
		RestConfig:     config,
		KubeConfigPath: kubeconfig,
	}, nil
}

// returnKubeConfigPath generates the path in the filesystem to kubeconfig
func returnKubeConfigPath(kubeConfigPath string) string {
	var kubeconfig string
	// We expect kubeconfig to be available at ~/.kube/config
	// However, sometimes some people may use the env var $KUBECONFIG
	// to set the path to the active one - we will switch on that here
	//
	// It's also possible to pass in a path directly
	switch {
	case kubeConfigPath != "":
		kubeconfig = kubeConfigPath
	case os.Getenv("KUBECONFIG") != "":
		kubeconfig = os.Getenv("KUBECONFIG")
	default:
		kubeconfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}

	return kubeconfig
}

// writeKubeConfig generates the kubeconfig at the path specified in the filesystem for auth later on
// func WriteKubeConfig(kubeconfig string, kubeConfigPath string) error {
// 	// We expect kubeconfig to be passed us base64 encoded

// 	data, err := b64.StdEncoding.DecodeString(kubeconfig)
// 	if err := os.WriteFile(kubeConfigPath, []byte(data), 0666); err != nil {
// 		log.Info("Kubeconfig could not be written to file: \n %s", data)
// 		log.Fatal(err)
// 	}

// 	log.Info("Kubeconfig Written to file: \n %s", data)

// 	return err
// }
