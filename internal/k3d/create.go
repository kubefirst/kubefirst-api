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

	pkg "github.com/kubefirst/kubefirst-api/internal"
	"github.com/kubefirst/kubefirst-api/internal/gitClient"
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
		log.Info().Msg("error creating k3d cluster")
		log.Info().Msgf(" err: %s %s %s", errLineOne, errLineTwo, err)
		return err
	}

	time.Sleep(20 * time.Second)

	kConfigString, _, err := pkg.ExecShellReturnStrings(k3dClient, "kubeconfig", "get", clusterName)
	if err != nil {
		return err
	}

	err = os.WriteFile(kubeconfig, []byte(kConfigString), 0644)
	if err != nil {
		log.Error().Err(err).Msg("error updating config")
		return fmt.Errorf("error updating config")
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
		return err
	}

	time.Sleep(20 * time.Second)

	kConfigString, _, err := pkg.ExecShellReturnStrings(k3dClient, "kubeconfig", "get", clusterName)
	if err != nil {
		return err
	}

	err = os.WriteFile(kubeconfig, []byte(kConfigString), 0644)
	if err != nil {
		log.Error().Err(err).Msg("error updating config")
		return fmt.Errorf("error updating config")
	}

	return nil
}

// should tokens be a *GitopsDirectoryValues? does it matter
func PrepareGitRepositories(
	gitProvider string,
	clusterName string,
	clusterType string,
	DestinationGitopsRepoURL string,
	gitopsDir string,
	gitopsTemplateBranch string,
	gitopsTemplateURL string,
	DestinationMetaphorRepoURL string,
	k1Dir string,
	gitopsTokens *GitopsDirectoryValues,
	metaphorDir string,
	metaphorTokens *MetaphorTokenValues,
	gitProtocol string,
	removeAtlantis bool,
	gitopsRepoName string,
	metaphorRepoName string,
) error {
	log.Info().Msg("hekeei")
	//* clone the gitops-template repo
	gitopsRepo, err := gitClient.CloneRefSetMain(gitopsTemplateBranch, gitopsDir, gitopsTemplateURL)
	if err != nil {
		log.Panic().Msgf("error opening repo at: %s, err: %v", gitopsDir, err)
	}
	log.Info().Msg("gitops repository clone complete")

	// * adjust the content for the gitops repo
	err = AdjustGitopsRepo(CloudProvider, clusterName, clusterType, gitopsDir, gitProvider, k1Dir, removeAtlantis, true)
	if err != nil {
		log.Info().Msgf("err: %v", err)
		return err
	}

	// * detokenize the gitops repo
	detokenizeGitGitops(gitopsDir, gitopsTokens, gitProtocol)
	if err != nil {
		return err
	}

	// * add new remote
	err = gitClient.AddRemote(DestinationGitopsRepoURL, gitProvider, gitopsRepo)
	if err != nil {
		return err
	}

	// ! metaphor
	// * adjust the content for the gitops repo

	err = AdjustMetaphorRepo(DestinationMetaphorRepoURL, gitopsDir, gitProvider, gitopsRepoName, metaphorRepoName,k1Dir)
	if err != nil {
		return err
	}

	// * detokenize the gitops repo
	detokenizeGitMetaphor(metaphorDir, metaphorTokens)
	if err != nil {
		return err
	}

	metaphorRepo, _ := git.PlainOpen(metaphorDir)
	//* commit initial gitops-template content
	err = gitClient.Commit(metaphorRepo, "committing initial detokenized metaphor repo content")
	if err != nil {
		return err
	}

	// * add new remote
	err = gitClient.AddRemote(DestinationMetaphorRepoURL, gitProvider, metaphorRepo)
	if err != nil {
		return err
	}

	// * commit initial gitops-template content
	// commit after metaphor content has been removed from gitops
	err = gitClient.Commit(gitopsRepo, "committing initial detokenized gitops-template repo content")
	if err != nil {
		return err
	}

	return nil
}

func PostRunPrepareGitopsRepository(clusterName string,
	//destinationGitopsRepoGitURL string,
	gitopsDir string,
	//gitopsRepo *git.Repository,
	tokens *GitopsDirectoryValues,
) error {

	err := postRunDetokenizeGitGitops(gitopsDir, tokens)
	if err != nil {
		return err
	}
	return nil
}
