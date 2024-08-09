/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"context"
	"fmt"

	"github.com/kubefirst/kubefirst-api/internal/aws"
	kube "github.com/kubefirst/kubefirst-api/internal/kubernetes"
	providerConfig "github.com/kubefirst/kubefirst-api/pkg/providerConfigs"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
)

func BootstrapAWSMgmtCluster(
	clientset kubernetes.Interface,
	cl *pkgtypes.Cluster,
	destinationGitopsRepoURL string,
	awsClient *aws.Configuration,
) error {
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
	if err := providerConfig.BootstrapSecrets(clientset, cl); err != nil {
		log.Error().Msgf("unable to bootstrap secrets: %s", err)
		return fmt.Errorf("unable to bootstrap secrets: %w", err)
	}

	// flag out the ecr token
	if cl.ECR {
		ecrToken, err := awsClient.GetECRAuthToken()
		if err != nil {
			return fmt.Errorf("error getting ecr token: %w", err)
		}

		dockerConfigString := fmt.Sprintf(`{"auths": {"%s": {"auth": "%s"}}}`, fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", cl.AWSAccountID, cl.CloudRegion), ecrToken)

		createSecrets := []kube.Secret{{
			Name:      "docker-config",
			Namespace: "argo",
			Contents:  map[string]string{"config.json": dockerConfigString},
		}}

		if err := kube.CreateSecretsIfNotExist(context.Background(), clientset, createSecrets); err != nil {
			log.Error().Msgf("error creating secrets: %s", err)
			return fmt.Errorf("error creating secrets: %w", err)
		}
	}

	return nil
}
