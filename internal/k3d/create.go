/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"fmt"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/rs/zerolog/log"

	pkg "github.com/konstructio/kubefirst-api/internal"
	"github.com/konstructio/kubefirst-api/internal/gitClient"
)

const (
	// https://hub.docker.com/r/rancher/k3s/tags?page=1&name=v1.23
	k3dImageTag string = "v1.26.3-k3s1"
)

// ClusterCreate create an k3d cluster
func ClusterCreate(clusterName string, k1Dir string, k3dClient string, kubeconfig string) error {
	log.Info().Msg("creating K3d cluster...")

	volumeDir := fmt.Sprintf("%s/minio-storage", k1Dir)
	if _, err := os.Stat(volumeDir); os.IsNotExist(err) {
		err := os.MkdirAll(volumeDir, os.ModePerm)
		if err != nil {
			log.Info().Msgf("%s directory already exists, continuing", volumeDir)
		}
	}
	errLineOne, errLineTwo, err := pkg.ExecShellReturnStrings(k3dClient, "cluster", "create",
		clusterName,
		"--image", fmt.Sprintf("rancher/k3s:%s", k3dImageTag),
		"--agents", "3",
		"--agents-memory", "1024m",
		"--registry-create", "k3d-"+clusterName+"-registry",
		"--k3s-arg", `--kubelet-arg=eviction-hard=imagefs.available<1%,nodefs.available<1%@agent:*`,
		"--k3s-arg", `--kubelet-arg=eviction-minimum-reclaim=imagefs.available=1%,nodefs.available=1%@agent:*`,
		"--volume", volumeDir+":/var/lib/rancher/k3s/storage@all",
		"--port", "443:443@loadbalancer",
	)
	if err != nil {
		log.Error().Msgf("error creating k3d cluster: %s %s %s", errLineOne, errLineTwo, err)
		return fmt.Errorf("error creating k3d cluster: %s %s %w", errLineOne, errLineTwo, err)
	}

	time.Sleep(20 * time.Second)

	kConfigString, _, err := pkg.ExecShellReturnStrings(k3dClient, "kubeconfig", "get", clusterName)
	if err != nil {
		return fmt.Errorf("error getting kubeconfig: %w", err)
	}

	err = os.WriteFile(kubeconfig, []byte(kConfigString), 0o644)
	if err != nil {
		log.Error().Err(err).Msg("error updating config")
		return fmt.Errorf("error updating config: %w", err)
	}

	return nil
}

// ClusterCreate create an k3d cluster for use with console and api
func ClusterCreateConsoleAPI(clusterName string, k1Dir string, k3dClient string, kubeconfig string) error {
	log.Info().Msg("creating K3d cluster...")

	_, _, err := pkg.ExecShellReturnStrings(k3dClient, "cluster", "create",
		clusterName,
		"--image", fmt.Sprintf("rancher/k3s:%s", k3dImageTag),
		"--agents", "1",
		"--agents-memory", "2048m",
		"--registry-create", "k3d-"+clusterName+"-registry",
		"--k3s-arg", `--kubelet-arg=eviction-hard=imagefs.available<1%,nodefs.available<1%@agent:*`,
		"--k3s-arg", `--kubelet-arg=eviction-minimum-reclaim=imagefs.available=1%,nodefs.available=1%@agent:*`,
		"--port", "443:443@loadbalancer",
		"--volume", k1Dir+":/.k1",
	)
	if err != nil {
		log.Info().Msg("error creating k3d cluster")
		return fmt.Errorf("error creating k3d cluster: %w", err)
	}

	time.Sleep(20 * time.Second)

	kConfigString, _, err := pkg.ExecShellReturnStrings(k3dClient, "kubeconfig", "get", clusterName)
	if err != nil {
		return fmt.Errorf("error getting kubeconfig: %w", err)
	}

	err = os.WriteFile(kubeconfig, []byte(kConfigString), 0o644)
	if err != nil {
		log.Error().Err(err).Msg("error updating config")
		return fmt.Errorf("error updating config: %w", err)
	}

	return nil
}

// should tokens be a *GitopsDirectoryValues? does it matter
func PrepareGitRepositories(
	gitProvider string,
	clusterName string,
	clusterType string,
	destinationGitopsRepoURL string,
	gitopsDir string,
	gitopsTemplateBranch string,
	gitopsTemplateURL string,
	destinationMetaphorRepoURL string,
	k1Dir string,
	gitopsTokens *GitopsDirectoryValues,
	metaphorDir string,
	metaphorTokens *MetaphorTokenValues,
	gitProtocol string,
	removeAtlantis bool,
) error {
	// * clone the gitops-template repo
	gitopsRepo, err := gitClient.CloneRefSetMain(gitopsTemplateBranch, gitopsDir, gitopsTemplateURL)
	if err != nil {
		log.Error().Msgf("error opening repo at: %s, err: %v", gitopsDir, err)
		return fmt.Errorf("error opening repo at: %s, err: %w", gitopsDir, err)
	}

	log.Info().Msg("gitops repository clone complete")

	// * adjust the content for the gitops repo
	err = AdjustGitopsRepo(CloudProvider, clusterName, clusterType, gitopsDir, gitProvider, removeAtlantis, true)
	if err != nil {
		log.Info().Msgf("err: %v", err)
		return err
	}

	// * detokenize the gitops repo
	err = detokenizeGitGitops(gitopsDir, gitopsTokens, gitProtocol)
	if err != nil {
		return err
	}

	// * add new remote
	err = gitClient.AddRemote(destinationGitopsRepoURL, gitProvider, gitopsRepo)
	if err != nil {
		return fmt.Errorf("error adding remote: %w", err)
	}

	// ! metaphor
	// * adjust the content for the gitops repo
	err = AdjustMetaphorRepo(destinationMetaphorRepoURL, gitopsDir, gitProvider, k1Dir)
	if err != nil {
		return err
	}

	// * detokenize the gitops repo
	err = detokenizeGitMetaphor(metaphorDir, metaphorTokens)
	if err != nil {
		return err
	}

	metaphorRepo, err := git.PlainOpen(metaphorDir)
	if err != nil {
		return fmt.Errorf("error opening metaphor repo: %w", err)
	}

	// * commit initial gitops-template content
	err = gitClient.Commit(metaphorRepo, "committing initial detokenized metaphor repo content")
	if err != nil {
		return fmt.Errorf("error committing initial detokenized metaphor repo content: %w", err)
	}

	// * add new remote
	err = gitClient.AddRemote(destinationMetaphorRepoURL, gitProvider, metaphorRepo)
	if err != nil {
		return fmt.Errorf("error adding remote: %w", err)
	}

	// * commit initial gitops-template content
	// commit after metaphor content has been removed from gitops
	err = gitClient.Commit(gitopsRepo, "committing initial detokenized gitops-template repo content")
	if err != nil {
		return fmt.Errorf("error committing initial detokenized gitops-template repo content: %w", err)
	}

	return nil
}

func PostRunPrepareGitopsRepository(gitopsDir string) error {
	if err := postRunDetokenizeGitGitops(gitopsDir); err != nil {
		return fmt.Errorf("error walking path %q: %w", gitopsDir, err)
	}
	return nil
}
