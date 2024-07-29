/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"context"
	"fmt"
	"time"	

	argocdapi "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	awsext "github.com/kubefirst/kubefirst-api/extensions/aws"
	pkg "github.com/kubefirst/kubefirst-api/internal"
	"github.com/kubefirst/kubefirst-api/internal/argocd"
	"github.com/kubefirst/kubefirst-api/internal/k8s"
	"github.com/kubefirst/kubefirst-api/internal/secrets"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	log "github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// InstallArgoCD
func (clctrl *ClusterController) InstallArgoCD() error {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.ArgoCDInstallCheck {

		var kcfg *k8s.KubernetesClient

		switch clctrl.CloudProvider {
		case "aws":
			kcfg = awsext.CreateEKSKubeconfig(&clctrl.AwsClient.Config, clctrl.ClusterName)
		case "akamai", "civo", "digitalocean", "k3s", "vultr":
			kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
		case "google":
			kcfg, err = clctrl.GoogleClient.GetContainerClusterAuth(clctrl.ClusterName, []byte(clctrl.GoogleAuth.KeyFile))
			if err != nil {
				return err
			}
		}

		argoCDInstallPath := fmt.Sprintf("github.com:kubefirst/manifests/argocd/cloud?ref=%s", pkg.KubefirstManifestRepoRef)
		log.Info().Msg("installing argocd")

		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.ArgoCDInstallStarted, "")
		err = argocd.ApplyArgoCDKustomize(kcfg.Clientset, argoCDInstallPath)
		if err != nil {
			telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.ArgoCDInstallFailed, err.Error())
			return err
		}

		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.ArgoCDInstallCompleted, "")

		// Wait for ArgoCD to be ready
		_, err = k8s.VerifyArgoCDReadiness(kcfg.Clientset, true, 300)
		if err != nil {
			log.Error().Msgf("error waiting for ArgoCD to become ready: %s", err)
			return err
		}

		clctrl.Cluster.ArgoCDInstallCheck = true
		err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return err
		}
	}

	return nil
}

// InitializeArgoCD
func (clctrl *ClusterController) InitializeArgoCD() error {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.ArgoCDInitializeCheck {
		var kcfg *k8s.KubernetesClient

		switch clctrl.CloudProvider {
		case "aws":
			kcfg = awsext.CreateEKSKubeconfig(&clctrl.AwsClient.Config, clctrl.ClusterName)
		case "akamai", "civo", "digitalocean", "k3s", "vultr":
			kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
		case "google":
			var err error
			kcfg, err = clctrl.GoogleClient.GetContainerClusterAuth(clctrl.ClusterName, []byte(clctrl.GoogleAuth.KeyFile))
			if err != nil {
				return err
			}
		}

		log.Info().Msg("Setting argocd username and password credentials")

		argocd.ArgocdSecretClient = kcfg.Clientset.CoreV1().Secrets("argocd")

		argocdPassword := k8s.GetSecretValue(argocd.ArgocdSecretClient, "argocd-initial-admin-secret", "password")
		if argocdPassword == "" {
			return fmt.Errorf("argocd password not found in secret")
		}

		log.Info().Msg("argocd username and password credentials set successfully")
		log.Info().Msg("getting an argocd auth token")

		var argoCDToken string

		switch clctrl.CloudProvider {
		case "aws", "civo", "google", "digitalocean", "vultr", "k3s":

			// kcfg.Clientset.RbacV1().
			argoCDStopChannel := make(chan struct{}, 1)
			defer func() {
				close(argoCDStopChannel)
			}()
			k8s.OpenPortForwardPodWrapper(
				kcfg.Clientset,
				kcfg.RestConfig,
				"argocd-server",
				"argocd",
				8080,
				8080,
				argoCDStopChannel,
			)
			argoCDToken, err = argocd.GetArgoCDToken("admin", argocdPassword)
			if err != nil {
				return err
			}
		}

		log.Info().Msg("argocd admin auth token set")

		clctrl.Cluster.ArgoCDPassword = argocdPassword
		clctrl.Cluster.ArgoCDAuthToken = argoCDToken
		clctrl.Cluster.ArgoCDInitializeCheck = true

		err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return err
		}
	}

	return nil
}

func RestartDeployment(ctx context.Context,clientset kubernetes.Interface,namespace string,deployment_name string) error {

	deploy,err := clientset.AppsV1().Deployments(namespace).Get(ctx,deployment_name,metav1.GetOptions{})
	
	if err != nil {
		return err
	}

	if deploy.Spec.Template.ObjectMeta.Annotations == nil {
        deploy.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
    }

	deploy.Spec.Template.ObjectMeta.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	_,err = clientset.AppsV1().Deployments(namespace).Update(ctx,deploy,metav1.UpdateOptions{})
	
	if err != nil {
		return fmt.Errorf("unable to update deployment %q: %w", deploy, err)
	}

	return nil
}

// DeployRegistryApplication
func (clctrl *ClusterController) DeployRegistryApplication() error {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.ArgoCDCreateRegistryCheck {
		var kcfg *k8s.KubernetesClient

		switch clctrl.CloudProvider {
		case "aws":
			kcfg = awsext.CreateEKSKubeconfig(&clctrl.AwsClient.Config, clctrl.ClusterName)
		case "akamai", "civo", "digitalocean", "k3s", "vultr":
			kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
		case "google":
			var err error
			kcfg, err = clctrl.GoogleClient.GetContainerClusterAuth(clctrl.ClusterName, []byte(clctrl.GoogleAuth.KeyFile))
			if err != nil {
				return err
			}
		}

		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.CreateRegistryStarted, "")
		argocdClient, err := argocdapi.NewForConfig(kcfg.RestConfig)
		if err != nil {
			return err
		}
		
		log.Info().Msg("applying the registry application to argocd")

		registryURL, err := clctrl.GetRepoURL()
		if err != nil {
			return err
		}

		var registryPath string
		if clctrl.CloudProvider == "k3d" {
			registryPath = fmt.Sprintf("registry/%s", clctrl.ClusterName)
		} else {
			registryPath = fmt.Sprintf("registry/clusters/%s", clctrl.ClusterName)
		}

		registryApplicationObject := argocd.GetArgoCDApplicationObject(
			registryURL,
			registryPath,
		)
		
		if clctrl.Kcfg==nil{
			clctrl.Kcfg = k8s.CreateKubeConfig(false,clctrl.ProviderConfig.Kubeconfig)
		}

		err = RestartDeployment(context.Background(),clctrl.Kcfg.Clientset,"argocd","argocd-applicationset-controller")
		
		if err!= nil {
			return err
		}

		log.Info().Msgf("successfull in restarting\n")

		retryAttempts := 2
		for attempt := 1; attempt <= retryAttempts; attempt++ {
			log.Info().Msgf("Attempt #%d to create Argo CD application...\n", attempt)

			app, err := argocdClient.ArgoprojV1alpha1().Applications("argocd").Create(context.Background(), registryApplicationObject, metav1.CreateOptions{})
			if err != nil {
				if attempt == retryAttempts {
					return err
				}
				log.Info().Msgf("Error creating Argo CD application on attempt number #%d: %v\n", attempt, err)
				time.Sleep(5 * time.Second)
				continue 
			}

			log.Info().Msgf("Argo CD application created successfully on attempt #%d: %s\n", attempt, app.Name)
			break 
		}


		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.CreateRegistryCompleted, "")

		clctrl.Cluster.ArgoCDCreateRegistryCheck = true
		err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return err
		}
	}

	return nil
}
