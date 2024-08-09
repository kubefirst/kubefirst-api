/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package google

import (
	"encoding/base64"
	"fmt"
	"net/http"

	container "cloud.google.com/go/container/apiv1"
	containerpb "cloud.google.com/go/container/apiv1/containerpb"
	"github.com/kubefirst/kubefirst-api/internal/k8s"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gocontainer "google.golang.org/api/container/v1"
	"google.golang.org/api/option"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ListContainerClusters
func (conf *Configuration) ListContainerClusters() (*containerpb.ListClustersResponse, error) {
	client, err := container.NewClusterManagerClient(conf.Context)
	if err != nil {
		return nil, fmt.Errorf("could not create google container client: %w", err)
	}

	clusters, err := client.ListClusters(conf.Context, &containerpb.ListClustersRequest{
		Parent: fmt.Sprintf("projects/%s/locations/-", conf.Project),
	})
	if err != nil {
		return nil, fmt.Errorf("error listing container clusters: %w", err)
	}

	return clusters, nil
}

// GetContainerCluster
func (conf *Configuration) GetContainerCluster(clusterName string) (*containerpb.Cluster, error) {
	client, err := container.NewClusterManagerClient(conf.Context)
	if err != nil {
		return nil, fmt.Errorf("could not create google container client: %w", err)
	}

	cluster, err := client.GetCluster(conf.Context, &containerpb.GetClusterRequest{
		Name: fmt.Sprintf("projects/%s/locations/%s/clusters/%s", conf.Project, conf.Region, clusterName),
	})
	if err != nil {
		return nil, fmt.Errorf("error getting container cluster: %w", err)
	}

	return cluster, nil
}

// GetContainerClusterAuth
func (conf *Configuration) GetContainerClusterAuth(clusterName string, keyFile []byte) (*k8s.KubernetesClient, error) {
	creds, err := google.CredentialsFromJSON(conf.Context, keyFile, gocontainer.CloudPlatformScope)
	if err != nil {
		return nil, fmt.Errorf("could not create google storage client credentials: %w", err)
	}

	client, err := container.NewClusterManagerClient(conf.Context, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("could not create google container client: %w", err)
	}

	cluster, err := client.GetCluster(conf.Context, &containerpb.GetClusterRequest{
		Name: fmt.Sprintf("projects/%s/locations/%s/clusters/%s", conf.Project, conf.Region, clusterName),
	})
	if err != nil {
		return nil, fmt.Errorf("error getting container cluster: %w", err)
	}

	// config := api.Config{
	// 	APIVersion: "v1",
	// 	Kind:       "Config",
	// 	Clusters:   map[string]*api.Cluster{},
	// 	AuthInfos:  map[string]*api.AuthInfo{},
	// 	Contexts:   map[string]*api.Context{},
	// }

	name := fmt.Sprintf("gke_%s_%s_%s", conf.Project, cluster.GetLocation(), cluster.GetName())
	cert, err := base64.StdEncoding.DecodeString(cluster.GetMasterAuth().GetClusterCaCertificate())
	if err != nil {
		return nil, fmt.Errorf("invalid certificate cluster=%s cert=%s: %w", name, cluster.GetMasterAuth().GetClusterCaCertificate(), err)
	}

	// Rest Config
	config := &rest.Config{
		Host: cluster.GetEndpoint(),
		TLSClientConfig: rest.TLSClientConfig{
			CAData: cert,
		},
	}
	config.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		return &oauth2.Transport{
			Source: creds.TokenSource,
			Base:   rt,
		}
	})

	// config.Clusters[name] = &api.Cluster{
	// 	CertificateAuthorityData: cert,
	// 	Server:                   "https://" + cluster.Endpoint,
	// }
	// config.Contexts[name] = &api.Context{
	// 	Cluster:  name,
	// 	AuthInfo: name,
	// }
	// config.AuthInfos[name] = &api.AuthInfo{
	// 	AuthProvider: &api.AuthProviderConfig{
	// 		Name: "google",
	// 		Config: map[string]string{
	// 			"scopes": "https://www.googleapis.com/auth/cloud-platform",
	// 		},
	// 	},
	// }
	//
	// var kubeConfig *rest.Config
	// var clientset *kubernetes.Clientset

	// for clusterName := range config.Clusters {
	// kubeConfig, err = clientcmd.NewNonInteractiveClientConfig(config, clusterName, &clientcmd.ConfigOverrides{CurrentContext: clusterName}, nil).ClientConfig()
	// if err != nil {
	// 	return nil, fmt.Errorf("error building kubernetes config: %w", err)
	// }

	// Client Set
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error buildling kubernetes clientset: %w", err)
	}
	// }

	return &k8s.KubernetesClient{
		Clientset:  clientset,
		RestConfig: config,
	}, nil
}
