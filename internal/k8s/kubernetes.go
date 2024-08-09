/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k8s

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	pkg "github.com/kubefirst/kubefirst-api/internal"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	coreV1Types "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type PatchJSON struct {
	Op   string `json:"op"`
	Path string `json:"path"`
}

func GetSecretValue(k8sClient coreV1Types.SecretInterface, secretName, key string) string {
	secret, err := k8sClient.Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("error getting key: %s from secret: %s", key, secretName)
	}
	return string(secret.Data[key])
}

// GetClientSet - Get reference to k8s credentials to use APIS
func GetClientSet(kubeconfigPath string) (*kubernetes.Clientset, error) {
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		log.Error().Err(err).Msg("Error getting kubeconfig")
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		log.Error().Err(err).Msg("Error getting clientset")
		return clientset, err
	}

	return clientset, nil
}

// GetClientConfig returns a rest.Config object for working with the Kubernetes
// API
func GetClientConfig(kubeconfigPath string) (*rest.Config, error) {
	clientconfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		log.Error().Err(err).Msg("Error getting kubeconfig")
		return nil, err
	}

	return clientconfig, nil
}

func WaitForNamespaceandPods(kubeconfigPath, kubectlClientPath, namespace, podLabel string) {
	if !viper.GetBool("create.softserve.ready") {
		x := 50
		for i := 0; i < x; i++ {
			_, _, err := pkg.ExecShellReturnStrings(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", namespace, "get", fmt.Sprintf("namespace/%s", namespace))
			if err != nil {
				log.Info().Msg(fmt.Sprintf("waiting for %s namespace to create ", namespace))
				time.Sleep(10 * time.Second)
			} else {
				log.Info().Msg(fmt.Sprintf("namespace %s found, continuing", namespace))
				time.Sleep(10 * time.Second)
				i = 51
			}
		}
		for i := 0; i < x; i++ {
			_, _, err := pkg.ExecShellReturnStrings(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", namespace, "get", "pods", "-l", podLabel)
			if err != nil {
				log.Info().Msg(fmt.Sprintf("waiting for %s pods to create ", namespace))
				time.Sleep(10 * time.Second)
			} else {
				log.Info().Msg(fmt.Sprintf("%s pods found, continuing", namespace))
				time.Sleep(10 * time.Second)
				break
			}
		}
		viper.Set("create.softserve.ready", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("soft-serve is ready, skipping")
	}
}
