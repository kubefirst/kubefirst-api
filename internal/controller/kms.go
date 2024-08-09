/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	githttps "github.com/go-git/go-git/v5/plumbing/transport/http"
	pkg "github.com/kubefirst/kubefirst-api/internal"
	"github.com/kubefirst/kubefirst-api/internal/gitClient"
	"github.com/kubefirst/kubefirst-api/internal/secrets"
)

// DetokenizeKMSKeyID
func (clctrl *ClusterController) DetokenizeKMSKeyID() error {
	cl, err := clctrl.GetCurrentClusterRecord()
	if err != nil {
		return err
	}

	if !cl.AWSKMSKeyDetokenizedCheck {
		if clctrl.CloudProvider != "aws" {
			return nil
		}

		// KMS
		gitopsRepo, err := git.PlainOpen(clctrl.ProviderConfig.GitopsDir)
		if err != nil {
			return err
		}
		awsKmsKeyID, err := clctrl.AwsClient.GetKmsKeyID(fmt.Sprintf("alias/vault_%s", clctrl.ClusterName))
		if err != nil {
			return err
		}

		clctrl.Cluster.AWSKMSKeyID = awsKmsKeyID
		err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return err
		}

		var registryPath string
		switch {
		case clctrl.CloudProvider == "civo" && clctrl.GitProvider == "github",
			clctrl.CloudProvider == "civo" && clctrl.GitProvider == "gitlab",
			clctrl.CloudProvider == "aws" && clctrl.GitProvider == "github",
			clctrl.CloudProvider == "aws" && clctrl.GitProvider == "gitlab",
			clctrl.CloudProvider == "google" && clctrl.GitProvider == "github",
			clctrl.CloudProvider == "google" && clctrl.GitProvider == "gitlab",
			clctrl.CloudProvider == "digitalocean" && clctrl.GitProvider == "github",
			clctrl.CloudProvider == "digitalocean" && clctrl.GitProvider == "gitlab",
			clctrl.CloudProvider == "vultr" && clctrl.GitProvider == "github",
			clctrl.CloudProvider == "vultr" && clctrl.GitProvider == "gitlab":
			registryPath = fmt.Sprintf("registry/clusters/%s", clctrl.ClusterName)
		default:
			registryPath = fmt.Sprintf("registry/%s", clctrl.ClusterName)
		}

		if err := pkg.ReplaceFileContent(
			fmt.Sprintf("%s/%s/components/vault/application.yaml", clctrl.ProviderConfig.GitopsDir, registryPath),
			"<AWS_KMS_KEY_ID>",
			awsKmsKeyID,
		); err != nil {
			return err
		}

		err = gitClient.Commit(gitopsRepo, "committing detokenized kms key")
		if err != nil {
			return err
		}

		err = gitopsRepo.Push(&git.PushOptions{
			RemoteName: clctrl.GitProvider,
			Auth: &githttps.BasicAuth{
				Username: clctrl.GitAuth.User,
				Password: clctrl.GitAuth.Token,
			},
		})
		if err != nil {
			return err
		}

		clctrl.Cluster.AWSKMSKeyDetokenizedCheck = true
		err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return err
		}
	}

	return nil
}
