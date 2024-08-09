/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kubefirst/kubefirst-api/configs"
)

// detokenizeGitGitops - Translate tokens by values on a given path
func detokenizeGitGitops(path string, tokens *GitopsDirectoryValues, gitProtocol string) error {
	err := filepath.Walk(path, detokenizeGitops(tokens, gitProtocol))
	if err != nil {
		return fmt.Errorf("error walking path %q: %w", path, err)
	}

	return nil
}

func detokenizeGitops(tokens *GitopsDirectoryValues, gitProtocol string) filepath.WalkFunc {
	return filepath.WalkFunc(func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fi.IsDir() {
			return nil
		}

		// ignore .git files
		if !strings.Contains(path, "/.git/") {
			read, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			// todo reduce to terraform tokens by moving to helm chart?
			newContents := string(read)
			newContents = strings.ReplaceAll(newContents, "<ALERTS_EMAIL>", "your@email.com")
			newContents = strings.ReplaceAll(newContents, "<ARGOCD_INGRESS_URL>", tokens.ArgocdIngressURL)
			newContents = strings.ReplaceAll(newContents, "<ARGO_WORKFLOWS_INGRESS_URL>", tokens.ArgoWorkflowsIngressURL)
			newContents = strings.ReplaceAll(newContents, "<ATLANTIS_ALLOW_LIST>", tokens.AtlantisAllowList)
			newContents = strings.ReplaceAll(newContents, "<ATLANTIS_INGRESS_URL>", tokens.AtlantisIngressURL)
			newContents = strings.ReplaceAll(newContents, "<CLUSTER_NAME>", tokens.ClusterName)
			newContents = strings.ReplaceAll(newContents, "<CLOUD_PROVIDER>", tokens.CloudProvider)
			newContents = strings.ReplaceAll(newContents, "<CLUSTER_ID>", tokens.ClusterID)
			newContents = strings.ReplaceAll(newContents, "<CLUSTER_TYPE>", tokens.ClusterType)
			newContents = strings.ReplaceAll(newContents, "<DOMAIN_NAME>", DomainName)
			newContents = strings.ReplaceAll(newContents, "<KUBEFIRST_TEAM>", tokens.KubefirstTeam)
			newContents = strings.ReplaceAll(newContents, "<KUBEFIRST_VERSION>", configs.K1Version)
			newContents = strings.ReplaceAll(newContents, "<KUBE_CONFIG_PATH>", tokens.KubeconfigPath)
			newContents = strings.ReplaceAll(newContents, "<METAPHOR_DEVELOPMENT_INGRESS_URL>", tokens.MetaphorDevelopmentIngressURL)
			newContents = strings.ReplaceAll(newContents, "<METAPHOR_STAGING_INGRESS_URL>", tokens.MetaphorStagingIngressURL)
			newContents = strings.ReplaceAll(newContents, "<METAPHOR_PRODUCTION_INGRESS_URL>", tokens.MetaphorProductionIngressURL)
			newContents = strings.ReplaceAll(newContents, "<GITHUB_HOST>", tokens.GithubHost)
			newContents = strings.ReplaceAll(newContents, "<GITHUB_OWNER>", strings.ToLower(tokens.GithubOwner))
			newContents = strings.ReplaceAll(newContents, "<GITHUB_USER>", tokens.GithubUser)
			newContents = strings.ReplaceAll(newContents, "<GIT_PROVIDER>", tokens.GitProvider)
			newContents = strings.ReplaceAll(newContents, "<GIT-PROTOCOL>", gitProtocol)
			newContents = strings.ReplaceAll(newContents, "<GITLAB_HOST>", tokens.GitlabHost)
			newContents = strings.ReplaceAll(newContents, "<GITLAB_OWNER>", tokens.GitlabOwner)
			newContents = strings.ReplaceAll(newContents, "<GITLAB_USER>", tokens.GitlabUser)
			newContents = strings.ReplaceAll(newContents, "<GITLAB_OWNER_GROUP_ID>", strconv.Itoa(tokens.GitlabOwnerGroupID))
			newContents = strings.ReplaceAll(newContents, "<VAULT_INGRESS_URL>", tokens.VaultIngressURL)
			newContents = strings.ReplaceAll(newContents, "<USE_TELEMETRY>", tokens.UseTelemetry)
			newContents = strings.ReplaceAll(newContents, "<K3D_DOMAIN>", DomainName)

			newContents = strings.ReplaceAll(newContents, "<GITOPS_REPO_URL>", tokens.GitopsRepoURL)

			// Switch the repo url based on https flag
			if gitProtocol == "https" {
				newContents = strings.ReplaceAll(newContents, "<GIT_FQDN>", fmt.Sprintf("https://%v.com/", tokens.GitProvider))
			} else {
				newContents = strings.ReplaceAll(newContents, "<GIT_FQDN>", fmt.Sprintf("git@%v.com:", tokens.GitProvider))
			}

			err = os.WriteFile(path, []byte(newContents), 0)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// postRunDetokenizeGitGitops - Translate tokens by values on a given path
func postRunDetokenizeGitGitops(path string) error {
	err := filepath.Walk(path, postRunDetokenizeGitops)
	if err != nil {
		return fmt.Errorf("error walking path: %w", err)
	}

	return nil
}

func postRunDetokenizeGitops(path string, fi os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if fi.IsDir() {
		return nil
	}

	// ignore .git files
	if !strings.Contains(path, "/.git/") {
		read, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("error reading file %q: %w", path, err)
		}

		// change Minio post cluster launch to cluster svc address
		read = bytes.ReplaceAll(read, []byte("https://minio."+DomainName), []byte("http://minio.minio.svc.cluster.local:9000"))
		if err := os.WriteFile(path, read, 0); err != nil {
			return fmt.Errorf("error writing file %q: %w", path, err)
		}
	}

	return nil
}

// detokenizeGitMetaphor - Translate tokens by values on a given path
func detokenizeGitMetaphor(path string, tokens *MetaphorTokenValues) error {
	if err := filepath.Walk(path, detokenize(tokens)); err != nil {
		return fmt.Errorf("error walking path %q: %w", path, err)
	}

	return nil
}

func detokenize(tokens *MetaphorTokenValues) filepath.WalkFunc {
	return filepath.WalkFunc(func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fi.IsDir() {
			return nil
		}

		// ignore .git files
		if !strings.Contains(path, "/.git/") {
			read, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			// todo reduce to terraform tokens by moving to helm chart?
			newContents := string(read)
			newContents = strings.ReplaceAll(newContents, "<METAPHOR_DEVELOPMENT_INGRESS_URL>", tokens.MetaphorDevelopmentIngressURL)
			newContents = strings.ReplaceAll(newContents, "<METAPHOR_STAGING_INGRESS_URL>", tokens.MetaphorStagingIngressURL)
			newContents = strings.ReplaceAll(newContents, "<METAPHOR_PRODUCTION_INGRESS_URL>", tokens.MetaphorProductionIngressURL)
			newContents = strings.ReplaceAll(newContents, "<CONTAINER_REGISTRY_URL>", tokens.ContainerRegistryURL) // todo need to fix metaphor repo
			newContents = strings.ReplaceAll(newContents, "<DOMAIN_NAME>", tokens.DomainName)
			newContents = strings.ReplaceAll(newContents, "<CLOUD_REGION>", tokens.CloudRegion)
			newContents = strings.ReplaceAll(newContents, "<CLUSTER_NAME>", tokens.ClusterName)

			err = os.WriteFile(path, []byte(newContents), 0)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
