/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package gitShim //nolint:revive // allowed during code reorg

import (
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/kubefirst/kubefirst-api/internal/gitClient"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/rs/zerolog/log"
)

// const (
// 	KubefirstGitHubOrganization      = "kubefirst"
// 	KubefirstGitopsCatalogRepository = "gitops-catalog"
// )

// PullWithAuth
func PullWithAuth(repo *git.Repository, remote string, branch string, auth transport.AuthMethod) error {
	w, _ := repo.Worktree()
	branchName := plumbing.NewBranchReferenceName(branch)
	err := w.Pull(&git.PullOptions{
		RemoteName:    remote,
		ReferenceName: branchName,
		Auth:          auth,
	})
	if err != nil {
		return fmt.Errorf("error during git pull: %w", err)
	}

	return nil
}

func PrepareMgmtCluster(cluster pkgtypes.Cluster) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal().Msgf("error getting home path: %s", err)
	}
	clusterDir := fmt.Sprintf("%s/.k1/%s", homeDir, cluster.ClusterName)
	gitopsDir := fmt.Sprintf("%s/.k1/%s/gitops", homeDir, cluster.ClusterName)

	if _, err := os.Stat(clusterDir); os.IsNotExist(err) {
		err := os.MkdirAll(clusterDir, 0o777)
		if err != nil {
			log.Fatal().Msgf("error creating home dir: %s", err)
			return err
		}
	}

	gitopsRepo, err := gitClient.CloneRefSetMain(cluster.GitopsTemplateBranch, gitopsDir, cluster.GitopsTemplateURL)
	if err != nil {
		log.Fatal().Msgf("error cloning repository: %s", err)

		return err
	}
	err = gitClient.AddRemote(fmt.Sprintf("https://%s/%s/gitops", cluster.GitHost, cluster.GitAuth.Owner), cluster.GitProvider, gitopsRepo)
	if err != nil {
		log.Fatal().Msgf("error cloning repository: %s", err)
		return err
	}

	return nil
}

func PrepareGitEnvironment(cluster *pkgtypes.Cluster, gitopsDir string) error {
	repoURL := fmt.Sprintf("https://%s/%s/gitops", cluster.GitHost, cluster.GitAuth.Owner)
	_, err := gitClient.ClonePrivateRepo("main", gitopsDir, repoURL, cluster.GitAuth.User, cluster.GitAuth.Token)
	if err != nil {
		log.Fatal().Msgf("error cloning repository: %s", err)

		return err
	}

	return nil
}

func PrepareGitOpsCatalog(gitopsCatalogDir string) error {
	repoURL := fmt.Sprintf("https://github.com/%s/%s", KubefirstGitHubOrganization, KubefirstGitopsCatalogRepository)
	_, err := gitClient.Clone("main", gitopsCatalogDir, repoURL)
	if err != nil {
		log.Fatal().Msgf("error cloning repository: %s", err)

		return err
	}

	return nil
}
