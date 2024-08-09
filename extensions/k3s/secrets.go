/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3s

import (
	"fmt"

	kube "github.com/kubefirst/kubefirst-api/internal/kubernetes"
	providerConfig "github.com/kubefirst/kubefirst-api/pkg/providerConfigs"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
)

func BootstrapK3sMgmtCluster(clientset kubernetes.Interface, cl *pkgtypes.Cluster, destinationGitopsRepoURL string) error {
	opts := providerConfig.BootstrapOptions{
		GitUser:                  cl.GitAuth.User,
		DestinationGitopsRepoURL: destinationGitopsRepoURL,
		GitProtocol:              cl.GitProtocol,
		CloudflareAPIToken:       cl.CloudflareAuth.APIToken,
		CloudAuth:                "",
		DNSProvider:              cl.DNSProvider,
		CloudProvider:            cl.CloudProvider,
		HTTPSPassword:            cl.GitAuth.Token,
		SSHToken:                 cl.GitAuth.PrivateKey,
	}

	if err := providerConfig.BootstrapMgmtCluster(clientset, opts); err != nil {
		log.Error().Msgf("unable to bootstrap management cluster: %s", err)
		return fmt.Errorf("unable to bootstrap management cluster: %w", err)
	}

	// Create secrets
	extraSecret := []kube.Secret{{
		Name:      "crossplane-secrets",
		Namespace: "crossplane-system",
		Contents: map[string]string{
			"username": cl.GitAuth.User,
			"password": cl.GitAuth.Token,
		},
	}}
	if err := providerConfig.BootstrapSecrets(clientset, cl, extraSecret...); err != nil {
		log.Error().Msgf("unable to bootstrap secrets: %s", err)
		return fmt.Errorf("unable to bootstrap secrets: %w", err)
	}

	return nil
}
