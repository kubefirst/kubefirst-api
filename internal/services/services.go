/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package services

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	v1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argocdapi "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	health "github.com/argoproj/gitops-engine/pkg/health"
	"github.com/go-git/go-git/v5"
	githttps "github.com/go-git/go-git/v5/plumbing/transport/http"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/gitShim"
	"github.com/kubefirst/kubefirst-api/internal/secrets"
	internalutils "github.com/kubefirst/kubefirst-api/internal/utils"
	"github.com/kubefirst/kubefirst-api/pkg/common"
	"github.com/kubefirst/kubefirst-api/pkg/providerConfigs"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	utils "github.com/kubefirst/kubefirst-api/pkg/utils"

	"github.com/kubefirst/kubefirst-api/internal/argocd"
	"github.com/kubefirst/kubefirst-api/internal/gitClient"
	"github.com/kubefirst/kubefirst-api/internal/k8s"
	"github.com/kubefirst/kubefirst-api/internal/vault"
	cp "github.com/otiai10/copy"
	log "github.com/rs/zerolog/log"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateService
func CreateService(cl *pkgtypes.Cluster, serviceName string, appDef *pkgtypes.GitopsCatalogApp, req *pkgtypes.GitopsCatalogAppCreateRequest, excludeArgoSync bool) error {
	switch cl.Status {
	case constants.ClusterStatusDeleted, constants.ClusterStatusDeleting, constants.ClusterStatusError, constants.ClusterStatusProvisioning:
		return fmt.Errorf("cluster %s - unable to deploy service %s to cluster: cannot deploy services to a cluster in %s state", cl.ClusterName, serviceName, cl.Status)
	}

	homeDir, _ := os.UserHomeDir()
	tmpGitopsDir := fmt.Sprintf("%s/.k1/%s/%s/gitops", homeDir, cl.ClusterName, serviceName)
	tmpGitopsCatalogDir := fmt.Sprintf("%s/.k1/%s/%s/gitops-catalog", homeDir, cl.ClusterName, serviceName)

	// Remove gitops dir
	err := os.RemoveAll(tmpGitopsDir)
	if err != nil {
		log.Fatal().Msgf("error removing gitops dir %s: %s", tmpGitopsDir, err)
		return err
	}

	// Remove gitops catalog dir
	err = os.RemoveAll(tmpGitopsCatalogDir)
	if err != nil {
		log.Fatal().Msgf("error removing gitops dir %s: %s", tmpGitopsCatalogDir, err)
		return err
	}

	err = gitShim.PrepareGitEnvironment(cl, tmpGitopsDir)
	if err != nil {
		log.Fatal().Msgf("an error occurred preparing git environment %s %s", tmpGitopsDir, err)
	}

	err = gitShim.PrepareGitOpsCatalog(tmpGitopsCatalogDir)
	if err != nil {
		log.Fatal().Msgf("an error occurred preparing gitops catalog environment %s %s", tmpGitopsDir, err)
	}

	gitopsRepo, _ := git.PlainOpen(tmpGitopsDir)

	clusterName := cl.ClusterName
	secretStoreRef := "vault-kv-secret"
	project := "default"
	clusterDestination := "in-cluster"
	environment := "mgmt"

	if req.WorkloadClusterName != "" {
		clusterName = req.WorkloadClusterName
		secretStoreRef = fmt.Sprintf("%s-vault-kv-secret", req.WorkloadClusterName)
		project = clusterName
		clusterDestination = clusterName
		environment = req.Environment
	}

	registryPath := getRegistryPath(clusterName, cl.CloudProvider, req.IsTemplate)

	clusterRegistryPath := fmt.Sprintf("%s/%s", tmpGitopsDir, registryPath)
	catalogServiceFolder := fmt.Sprintf("%s/%s", tmpGitopsCatalogDir, serviceName)

	kcfg := internalutils.GetKubernetesClient(cl.ClusterName)

	var fullDomainName string
	if cl.SubdomainName != "" {
		fullDomainName = fmt.Sprintf("%s.%s", cl.SubdomainName, cl.DomainName)
	} else {
		fullDomainName = cl.DomainName
	}

	vaultURL := fmt.Sprintf("https://vault.%s", fullDomainName)

	if cl.CloudProvider == "k3d" {
		vaultURL = "http://vault.vault.svc:8200"
	}

	// If there are secret values, create a vault secret
	if len(req.SecretKeys) > 0 {
		log.Info().Msgf("cluster %s - application %s has secrets, creating vault values", clusterName, appDef.Name)

		s := make(map[string]interface{}, 0)

		for _, secret := range req.SecretKeys {
			s[secret.Name] = secret.Value
		}

		// Get token
		existingKubernetesSecret, err := k8s.ReadSecretV2(kcfg.Clientset, vault.VaultNamespace, vault.VaultSecretName)
		if err != nil || existingKubernetesSecret == nil {
			return fmt.Errorf("cluster %s - error getting vault token: %s", clusterName, err)
		}

		vaultClient, err := vaultapi.NewClient(&vaultapi.Config{
			Address: vaultURL,
		})
		if err != nil {
			return fmt.Errorf("cluster %s - error initializing vault client: %s", clusterName, err)
		}

		vaultClient.SetToken(existingKubernetesSecret["root-token"])

		resp, err := vaultClient.KVv2("secret").Put(context.Background(), appDef.Name, s)
		if err != nil {
			return fmt.Errorf("cluster %s - error putting vault secret: %s", clusterName, err)
		}

		log.Info().Msgf("cluster %s - created vault secret data for application %s %s", clusterName, appDef.Name, resp.VersionMetadata.CreatedTime)
	}

	// Create service files in gitops dir
	err = gitShim.PullWithAuth(
		gitopsRepo,
		"origin",
		"main",
		&githttps.BasicAuth{
			Username: cl.GitAuth.User,
			Password: cl.GitAuth.Token,
		},
	)
	if err != nil {
		log.Warn().Msgf("cluster %s - error pulling gitops repo: %s", clusterName, err)
	}

	if !req.IsTemplate {
		// Create Tokens
		gitopsKubefirstTokens := utils.CreateTokensFromDatabaseRecord(cl, registryPath, secretStoreRef, project, clusterDestination, environment, clusterName)

		// Detokenize App Template
		err = providerConfigs.DetokenizeGitGitops(catalogServiceFolder, gitopsKubefirstTokens, cl.GitProtocol, cl.CloudflareAuth.OriginCaIssuerKey != "")
		if err != nil {
			return fmt.Errorf("cluster %s - error opening file: %s", clusterName, err)
		}

		// Detokenize Config Keys
		err = DetokenizeConfigKeys(catalogServiceFolder, req.ConfigKeys)
		if err != nil {
			return fmt.Errorf("cluster %s - error opening file: %s", clusterName, err)
		}
	}

	// Get Ingress links
	links := common.GetIngressLinks(catalogServiceFolder, fullDomainName)

	err = cp.Copy(catalogServiceFolder, clusterRegistryPath, cp.Options{})
	if err != nil {
		log.Error().Msgf("Error populating gitops repository with catalog components content: %s. error: %s", serviceName, err.Error())
		return err
	}

	// Commit to gitops repository
	err = gitClient.Commit(gitopsRepo, fmt.Sprintf("adding %s to the cluster %s on behalf of %s", serviceName, clusterName, req.User))
	if err != nil {
		return fmt.Errorf("cluster %s - error committing service file: %s", clusterName, err)
	}
	err = gitopsRepo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth: &githttps.BasicAuth{
			Username: cl.GitAuth.User,
			Password: cl.GitAuth.Token,
		},
		Force: true,
	})
	if err != nil {
		return fmt.Errorf("cluster %s - error pushing commit for service file: %s", clusterName, err)
	}

	existingService, _ := secrets.GetServices(kcfg.Clientset, clusterName)

	if existingService.ClusterName == "" {
		// Add to list
		err = secrets.CreateClusterServiceList(kcfg.Clientset, clusterName)
		if err != nil {
			return err
		}
	}

	// Update list
	err = secrets.InsertClusterServiceListEntry(kcfg.Clientset, clusterName, &pkgtypes.Service{
		Name:        serviceName,
		Default:     false,
		Description: appDef.Description,
		Image:       appDef.ImageURL,
		Links:       links,
		Status:      "",
		CreatedBy:   req.User,
	})
	if err != nil {
		return err
	}

	if excludeArgoSync || req.IsTemplate {
		return nil
	}

	// Wait for ArgoCD application sync
	argocdClient, err := argocdapi.NewForConfig(kcfg.RestConfig)
	if err != nil {
		return err
	}

	// Sync registry
	argoCDHost := fmt.Sprintf("https://argocd.%s", fullDomainName)
	if cl.CloudProvider == "k3d" {
		argoCDHost = "http://argocd-server.argocd.svc.cluster.local"
	}

	argoCDToken, err := argocd.GetArgocdTokenV2(argoCDHost, "admin", cl.ArgoCDPassword)
	if err != nil {
		log.Warn().Msgf("error getting argocd token: %s", err)
		return err
	}
	err = argocd.RefreshRegistryApplication(argoCDHost, argoCDToken)
	if err != nil {
		log.Warn().Msgf("error refreshing registry application: %s", err)
		return err
	}

	// Wait for app to be created
	for i := 0; i < 50; i++ {
		_, err := argocdClient.ArgoprojV1alpha1().Applications("argocd").Get(context.Background(), serviceName, v1.GetOptions{})
		if err != nil {
			log.Info().Msgf("cluster %s - waiting for app %s to be created", clusterName, serviceName)
			time.Sleep(time.Second * 10)
		} else {
			break
		}
		if i == 50 {
			return fmt.Errorf("cluster %s - error waiting for app %s to be created: %s", clusterName, serviceName, err)
		}
	}

	// Wait for app to be synchronized and healthy
	for i := 0; i < 50; i++ {
		if i == 50 {
			return fmt.Errorf("cluster %s - error waiting for app %s to synchronize: %s", clusterName, serviceName, err)
		}
		app, err := argocdClient.ArgoprojV1alpha1().Applications("argocd").Get(context.Background(), serviceName, v1.GetOptions{})
		if err != nil {
			return fmt.Errorf("cluster %s - error getting argocd application %s: %s", clusterName, serviceName, err)
		}
		if app.Status.Sync.Status == v1alpha1.SyncStatusCodeSynced && app.Status.Health.Status == health.HealthStatusHealthy {
			log.Info().Msgf("cluster %s - app %s synchronized", clusterName, serviceName)
			break
		}
		log.Info().Msgf("cluster %s - waiting for app %s to sync", clusterName, serviceName)
		time.Sleep(time.Second * 10)
	}

	return nil
}

// DeleteService
func DeleteService(cl *pkgtypes.Cluster, serviceName string, def pkgtypes.GitopsCatalogAppDeleteRequest) error {
	var gitopsRepo *git.Repository

	clusterName := cl.ClusterName

	if def.WorkloadClusterName != "" {
		clusterName = def.WorkloadClusterName
	}

	kcfg := internalutils.GetKubernetesClient(clusterName)

	// Remove from list
	svc, err := secrets.GetService(kcfg.Clientset, clusterName, serviceName)
	if err != nil {
		return fmt.Errorf("cluster %s - error finding service: %s", clusterName, err)
	}

	if !def.SkipFiles {
		homeDir, _ := os.UserHomeDir()
		tmpGitopsDir := fmt.Sprintf("%s/.k1/%s/%s/gitops", homeDir, cl.ClusterName, serviceName)

		// Remove gitops dir
		err = os.RemoveAll(tmpGitopsDir)
		if err != nil {
			log.Fatal().Msgf("error removing gitops dir %s: %s", tmpGitopsDir, err)
			return err
		}

		err = gitShim.PrepareGitEnvironment(cl, tmpGitopsDir)
		if err != nil {
			log.Fatal().Msgf("an error occurred preparing git environment %s %s", tmpGitopsDir, err)
		}

		gitopsRepo, _ = git.PlainOpen(tmpGitopsDir)

		registryPath := getRegistryPath(clusterName, cl.CloudProvider, def.IsTemplate)

		serviceFile := fmt.Sprintf("%s/%s/%s.yaml", tmpGitopsDir, registryPath, serviceName)
		componentsServiceFolder := fmt.Sprintf("%s/%s/components/%s", tmpGitopsDir, registryPath, serviceName)

		err = gitShim.PullWithAuth(
			gitopsRepo,
			cl.GitProvider,
			"main",
			&githttps.BasicAuth{
				Username: cl.GitAuth.User,
				Password: cl.GitAuth.Token,
			},
		)
		if err != nil {
			log.Warn().Msgf("cluster %s - error pulling gitops repo: %s", clusterName, err)
		}

		// removing registry service file
		_, err = os.Stat(serviceFile)
		if err != nil {
			return fmt.Errorf("file %s does not exist in repository", serviceFile)
		}

		err := os.Remove(serviceFile)
		if err != nil {
			return fmt.Errorf("cluster %s - error deleting file: %s", clusterName, err)
		}

		// removing components service folder
		_, err = os.Stat(componentsServiceFolder)
		if err != nil {
			return fmt.Errorf("folder %s does not exist in repository", componentsServiceFolder)
		}

		if err := os.RemoveAll(componentsServiceFolder); err != nil {
			return fmt.Errorf("cluster %s - error deleting components folder: %s", clusterName, err)
		}

		// Commit to gitops repository
		err = gitClient.Commit(gitopsRepo, fmt.Sprintf("removing %s from the cluster %s on behalf of %s", serviceName, clusterName, def.User))
		if err != nil {
			return fmt.Errorf("cluster %s - error deleting service file: %s", clusterName, err)
		}

		err = gitopsRepo.Push(&git.PushOptions{
			RemoteName: "origin",
			Auth: &githttps.BasicAuth{
				Username: cl.GitAuth.User,
				Password: cl.GitAuth.Token,
			},
		})
		if err != nil {
			return fmt.Errorf("cluster %s - error pushing commit for service file: %s", clusterName, err)
		}
	}

	err = secrets.DeleteClusterServiceListEntry(kcfg.Clientset, clusterName, &svc)
	if err != nil {
		return err
	}

	return nil
}

// ValidateService
func ValidateService(cl *pkgtypes.Cluster, serviceName string, def *pkgtypes.GitopsCatalogAppCreateRequest) (bool, error) {
	canDeleleteService := true

	var gitopsRepo *git.Repository

	clusterName := cl.ClusterName

	if def.WorkloadClusterName != "" {
		clusterName = def.WorkloadClusterName
	}

	homeDir, _ := os.UserHomeDir()
	tmpGitopsDir := fmt.Sprintf("%s/.k1/%s/%s/gitops", homeDir, cl.ClusterName, serviceName)

	// Remove gitops dir
	err := os.RemoveAll(tmpGitopsDir)
	if err != nil {
		log.Fatal().Msgf("error removing gitops dir %s: %s", tmpGitopsDir, err)
		return false, err
	}

	err = gitShim.PrepareGitEnvironment(cl, tmpGitopsDir)
	if err != nil {
		log.Fatal().Msgf("an error occurred preparing git environment %s %s", tmpGitopsDir, err)
	}

	gitopsRepo, _ = git.PlainOpen(tmpGitopsDir)

	registryPath := getRegistryPath(clusterName, cl.CloudProvider, def.IsTemplate)

	serviceFile := fmt.Sprintf("%s/%s/%s.yaml", tmpGitopsDir, registryPath, serviceName)

	err = gitShim.PullWithAuth(
		gitopsRepo,
		cl.GitProvider,
		"main",
		&githttps.BasicAuth{
			Username: cl.GitAuth.User,
			Password: cl.GitAuth.Token,
		},
	)
	if err != nil {
		log.Warn().Msgf("cluster %s - error pulling gitops repo: %s", clusterName, err)
	}

	// removing registry service file
	_, err = os.Stat(serviceFile)
	if err != nil {
		canDeleleteService = false
	}

	return canDeleleteService, nil
}

// AddDefaultServices
func AddDefaultServices(cl *pkgtypes.Cluster) error {
	kcfg := internalutils.GetKubernetesClient(cl.ClusterName)

	err := secrets.CreateClusterServiceList(kcfg.Clientset, cl.ClusterName)
	if err != nil {
		return err
	}

	var fullDomainName string
	if cl.SubdomainName != "" {
		fullDomainName = fmt.Sprintf("%s.%s", cl.SubdomainName, cl.DomainName)
	} else {
		fullDomainName = cl.DomainName
	}

	defaults := []pkgtypes.Service{
		{
			Name:        cl.GitProvider,
			Default:     true,
			Description: "The git repositories contain all the Infrastructure as Code and Gitops configurations.",
			Image:       fmt.Sprintf("https://assets.kubefirst.com/console/%s.svg", cl.GitProvider),
			Links: []string{
				fmt.Sprintf("https://%s/%s/gitops", cl.GitHost, cl.GitAuth.Owner),
				fmt.Sprintf("https://%s/%s/metaphor", cl.GitHost, cl.GitAuth.Owner),
			},
			Status:    "",
			CreatedBy: "kbot",
		},
		{
			Name:        "Vault",
			Default:     true,
			Description: "Kubefirst's secrets manager and identity provider.",
			Image:       "https://assets.kubefirst.com/console/vault.svg",
			Links:       []string{fmt.Sprintf("https://vault.%s", fullDomainName)},
			Status:      "",
			CreatedBy:   "kbot",
		},
		{
			Name:        "Argo CD",
			Default:     true,
			Description: "A Gitops oriented continuous delivery tool for managing all of our applications across our Kubernetes clusters.",
			Image:       "https://assets.kubefirst.com/console/argocd.svg",
			Links:       []string{fmt.Sprintf("https://argocd.%s", fullDomainName)},
			Status:      "",
			CreatedBy:   "kbot",
		},
		{
			Name:        "Argo Workflows",
			Default:     true,
			Description: "The workflow engine for orchestrating parallel jobs on Kubernetes.",
			Image:       "https://assets.kubefirst.com/console/argocd.svg",
			Links:       []string{fmt.Sprintf("https://argo.%s/workflows", fullDomainName)},
			Status:      "",
			CreatedBy:   "kbot",
		},
		{
			Name:        "Atlantis",
			Default:     true,
			Description: "Kubefirst manages Terraform workflows with Atlantis automation.",
			Image:       "https://assets.kubefirst.com/console/atlantis.svg",
			Links:       []string{fmt.Sprintf("https://atlantis.%s", fullDomainName)},
			Status:      "",
			CreatedBy:   "kbot",
		},
		{
			Name:        "Metaphor",
			Default:     true,
			Description: "A multi-environment demonstration space for frontend application best practices that's easy to apply to other projects.",
			Image:       "https://assets.kubefirst.com/console/metaphor.svg",
			Links: []string{
				fmt.Sprintf("https://metaphor-development.%s", fullDomainName),
				fmt.Sprintf("https://metaphor-staging.%s", fullDomainName),
				fmt.Sprintf("https://metaphor-production.%s", fullDomainName),
			},
			Status:    "",
			CreatedBy: "kbot",
		},
	}

	for _, svc := range defaults {
		err := secrets.InsertClusterServiceListEntry(kcfg.Clientset, cl.ClusterName, &svc)
		if err != nil {
			return err
		}
	}

	return nil
}

func DetokenizeConfigKeys(serviceFilePath string, configKeys []pkgtypes.GitopsCatalogAppKeys) error {
	return filepath.Walk(serviceFilePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			for _, configKey := range configKeys {
				data = bytes.ReplaceAll(data, []byte(configKey.Name), []byte(configKey.Value))
			}

			err = os.WriteFile(path, data, 0)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func getRegistryPath(clusterName string, cloudProvider string, isTemplate bool) string {
	if isTemplate && cloudProvider != "k3d" {
		return filepath.Join("templates", clusterName)
	}

	if cloudProvider == "k3d" {
		return filepath.Join("clusters", clusterName)
	}

	return filepath.Join("registry", "clusters", clusterName)
}
