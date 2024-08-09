/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package providerConfigs //nolint:revive,stylecheck // allowing temporarily for better code organization

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	pkg "github.com/kubefirst/kubefirst-api/internal"
	"github.com/kubefirst/kubefirst-api/internal/gitClient"
	cp "github.com/otiai10/copy"
	"github.com/rs/zerolog/log"
)

const (
	AkamaiGitHub       = "akamai-github"
	AwsGitHub          = "aws-github"
	AwsGitLab          = "aws-gitlab"
	CivoGitHub         = "civo-github"
	CivoGitLab         = "civo-gitlab"
	GoogleGitHub       = "google-github"
	GoogleGitLab       = "google-gitlab"
	DigitalOceanGitHub = "digitalocean-github"
	DigitalOceanGitLab = "digitalocean-gitlab"
	VultrGitHub        = "vultr-github"
	VultrGitLab        = "vultr-gitlab"
	K3sGitHub          = "k3s-github"
	K3sGitLab          = "k3s-gitlab"
)

func removeAllWithLogger(path string) {
	if err := os.RemoveAll(path); err != nil {
		// allowing the skip of errors here
		log.Error().Msgf("Error removing %q from filesystem: %s", path, err.Error())
	}
}

func adjustGitOpsRepoForProvider(cloudProvider, gitProvider, gitopsRepoDir, clusterType, clusterName string, apexContentExists, isK3D bool) error {
	opt := cp.Options{
		Skip: func(src string) (bool, error) {
			if strings.HasSuffix(src, ".git") {
				return true, nil
			} else if strings.Index(src, "/.terraform") > 0 {
				return true, nil
			}
			// Add more stuff to be ignored here
			return false, nil
		},
	}
	driverContent := fmt.Sprintf("%s/%s-%s/", gitopsRepoDir, cloudProvider, gitProvider)

	if err := cp.Copy(driverContent, gitopsRepoDir, opt); err != nil {
		log.Error().Msgf("Error populating gitops repository with driver content: %s. error: %s", fmt.Sprintf("%s-%s", cloudProvider, gitProvider), err.Error())
		return fmt.Errorf("error populating gitops repository with driver content: %s. error: %w", fmt.Sprintf("%s-%s", cloudProvider, gitProvider), err)
	}

	if err := os.RemoveAll(driverContent); err != nil {
		log.Error().Msgf("Error removing %q from filesystem: %s", driverContent, err.Error())
		return fmt.Errorf("error removing %q from filesystem: %w", driverContent, err)
	}

	// * copy $HOME/.k1/gitops/templates/${clusterType}/* $HOME/.k1/gitops/registry/${clusterName}
	clusterContent := filepath.Join(gitopsRepoDir, "templates", clusterType)

	// Remove apex content if apex content already exists
	if apexContentExists {
		log.Warn().Msgf("removing nginx-apex since apexContentExists was %v", apexContentExists)
		if err := os.Remove(filepath.Join(clusterContent, "nginx-apex.yaml")); err != nil {
			log.Error().Msgf("Error removing %q from filesystem: %s", filepath.Join(clusterContent, "nginx-apex.yaml"), err.Error())
			return fmt.Errorf("error removing %q from filesystem: %w", filepath.Join(clusterContent, "nginx-apex.yaml"), err)
		}
		if err := os.RemoveAll(filepath.Join(clusterContent, "nginx-apex")); err != nil {
			log.Error().Msgf("Error removing %q from filesystem: %s", filepath.Join(clusterContent, "nginx-apex"), err.Error())
			return fmt.Errorf("error removing %q from filesystem: %w", filepath.Join(clusterContent, "nginx-apex"), err)
		}
	} else {
		log.Warn().Msgf("will create nginx-apex since apexContentExists was %v", apexContentExists)
	}

	location := filepath.Join(gitopsRepoDir, "registry", "clusters", clusterName)
	if isK3D {
		location = filepath.Join(gitopsRepoDir, "registry", clusterName)
	}

	if err := cp.Copy(clusterContent, location, opt); err != nil {
		log.Info().Msgf("Error populating cluster content with %s. error: %s", clusterContent, err.Error())
		return fmt.Errorf("error populating cluster content with %s: %w", clusterContent, err)
	}

	if err := os.RemoveAll(filepath.Join(gitopsRepoDir, "templates", "mgmt")); err != nil {
		log.Error().Msgf("Error removing %q from filesystem: %s", filepath.Join(gitopsRepoDir, "templates", "mgmt"), err.Error())
		return fmt.Errorf("error removing %q from filesystem: %w", filepath.Join(gitopsRepoDir, "templates", "mgmt"), err)
	}

	return nil
}

// AdjustGitopsRepo
func AdjustGitopsRepo(
	cloudProvider string,
	clusterName string,
	clusterType string,
	gitopsRepoDir string,
	gitProvider string,
	apexContentExists bool,
	useCloudflareOriginIssuer bool,
) error {
	// * clean up all other platforms
	for _, platform := range pkg.SupportedPlatforms {
		if platform != fmt.Sprintf("%s-%s", cloudProvider, gitProvider) {
			removeAllWithLogger(filepath.Join(gitopsRepoDir, platform))
		}
	}

	if !useCloudflareOriginIssuer {
		removeAllWithLogger(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/mgmt/cloudflare-origin-ca-issuer.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
		removeAllWithLogger(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/mgmt/cloudflare-origin-issuer-crd.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
		removeAllWithLogger(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/mgmt/components/argo-workflows/cloudflareissuer.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
		removeAllWithLogger(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/mgmt/components/argocd/cloudflareissuer.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
		removeAllWithLogger(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/mgmt/components/atlantis/cloudflareissuer.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
		removeAllWithLogger(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/mgmt/components/chartmuseum/cloudflareissuer.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
		removeAllWithLogger(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/mgmt/components/kubefirst/cloudflareissuer.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
		removeAllWithLogger(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/mgmt/components/vault/cloudflareissuer.yaml", gitopsRepoDir, cloudProvider, gitProvider)))

		removeAllWithLogger(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/workload-cluster/cloudflare-origin-issuer", gitopsRepoDir, cloudProvider, gitProvider)))
		removeAllWithLogger(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/workload-cluster/40-cloudflare-origin-issuer-crd.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
		removeAllWithLogger(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/workload-cluster/41-cloudflare-origin-ca-issuer.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
		removeAllWithLogger(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/workload-cluster/45-cloudflare-origin-issuer.yaml", gitopsRepoDir, cloudProvider, gitProvider)))

		removeAllWithLogger(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/workload-vcluster/cloudflare-origin-issuer", gitopsRepoDir, cloudProvider, gitProvider)))
		removeAllWithLogger(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/workload-vcluster/40-cloudflare-origin-issuer-crd.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
		removeAllWithLogger(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/workload-vcluster/41-cloudflare-origin-ca-issuer.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
		removeAllWithLogger(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/workload-vcluster/45-cloudflare-origin-issuer.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
	}

	cloudAndGitProvider := strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider))

	switch cloudAndGitProvider {
	case AkamaiGitHub,
		AwsGitHub,
		AwsGitLab,
		CivoGitHub,
		CivoGitLab,
		GoogleGitHub,
		GoogleGitLab,
		DigitalOceanGitHub,
		DigitalOceanGitLab,
		VultrGitHub,
		VultrGitLab,
		K3sGitHub,
		K3sGitLab:
		return adjustGitOpsRepoForProvider(cloudProvider, gitProvider, gitopsRepoDir, clusterType, clusterName, apexContentExists, false)

	default:
		return adjustGitOpsRepoForProvider(cloudProvider, gitProvider, gitopsRepoDir, clusterType, clusterName, apexContentExists, true)
	}
}

func copyContents(source, destination string, createPath bool) error {
	opt := cp.Options{
		Skip: func(src string) (bool, error) {
			if strings.HasSuffix(src, ".git") {
				return true, nil
			} else if strings.Index(src, "/.terraform") > 0 {
				return true, nil
			}
			// Add more stuff to be ignored here
			return false, nil
		},
	}

	if createPath {
		target := filepath.Dir(destination)
		if err := os.MkdirAll(target, 0o700); err != nil {
			log.Error().Msgf("Error creating directory at %s. error: %s", target, err.Error())
			return fmt.Errorf("error creating directory at %s: %w", target, err)
		}
	}

	log.Info().Msgf("copying %q to %q", source, destination)
	if err := cp.Copy(source, destination, opt); err != nil {
		log.Error().Msgf("unable to copy %q to %q: %s", source, destination, err.Error())
		return fmt.Errorf("error copying %q to %q: %w", source, destination, err)
	}

	return nil
}

// AdjustMetaphorRepo
func AdjustMetaphorRepo(
	destinationMetaphorRepoURL string,
	gitopsRepoDir string,
	gitProvider string,
	cloudProvider string,
	k1Dir string,
) error {
	// * create ~/.k1/metaphor
	metaphorDir := filepath.Join(k1Dir, "metaphor")
	if err := os.Mkdir(metaphorDir, 0o700); err != nil {
		log.Error().Msgf("Error creating metaphor directory at %s. error: %s", metaphorDir, err.Error())
		return fmt.Errorf("error creating metaphor directory at %s: %w", metaphorDir, err)
	}

	// * git init
	metaphorRepo, err := git.PlainInit(metaphorDir, false)
	if err != nil {
		log.Error().Msgf("Error initializing git repository at %s. error: %s", metaphorDir, err.Error())
		return fmt.Errorf("error initializing git repository at %s: %w", metaphorDir, err)
	}

	switch fmt.Sprintf("%s-%s", cloudProvider, gitProvider) {
	case AkamaiGitHub,
		AwsGitHub,
		AwsGitLab,
		CivoGitHub,
		CivoGitLab:

		// * metaphor app source
		metaphorContent := filepath.Join(gitopsRepoDir, "metaphor")
		if err := copyContents(metaphorContent, metaphorDir, false); err != nil {
			return err
		}

		// Remove metaphor content from gitops repository directory
		if err := os.RemoveAll(metaphorContent); err != nil {
			log.Error().Msgf("Error removing %q from filesystem: %s", metaphorContent, err.Error())
			return fmt.Errorf("error removing %q from filesystem: %w", metaphorContent, err)
		}
	default:
		// * remove .argo and .github folders
		var (
			dotArgoFolder   = filepath.Join(metaphorDir, ".argo")
			dotGitHubFolder = filepath.Join(metaphorDir, ".github")
		)

		if err := os.RemoveAll(dotArgoFolder); err != nil {
			log.Error().Msgf("Error removing %q from filesystem: %s", dotArgoFolder, err.Error())
			return fmt.Errorf("error removing %q from filesystem: %w", dotArgoFolder, err)
		}

		if err := os.RemoveAll(dotGitHubFolder); err != nil {
			log.Error().Msgf("Error removing %q from filesystem: %s", dotGitHubFolder, err.Error())
			return fmt.Errorf("error removing %q from filesystem: %w", dotGitHubFolder, err)
		}

		// * copy ci content
		if gitProvider == "github" || gitProvider == "gitlab" {
			var src, dest string

			if gitProvider == "github" {
				src = filepath.Join(k1Dir, "gitops", "ci", ".github")
				dest = filepath.Join(metaphorDir, ".github")
			} else {
				src = filepath.Join(k1Dir, "gitops", "ci", ".gitlab-ci.yml")
				dest = filepath.Join(metaphorDir, ".gitlab-ci.yml")
			}

			if err := copyContents(src, dest, true); err != nil {
				return fmt.Errorf("error copying CI content: %w", err)
			}
		}

		// * metaphor app source
		metaphorLocation := filepath.Join(gitopsRepoDir, "metaphor")
		if err := copyContents(metaphorLocation, metaphorDir, false); err != nil {
			return fmt.Errorf("error copying metaphor app source: %w", err)
		}

		// * copy $HOME/.k1/gitops/ci/.argo/* $HOME/.k1/metaphor/.argo
		argoWorkflowsSource := filepath.Join(k1Dir, "gitops", "ci", ".argo")
		argoWorkflowsDest := filepath.Join(metaphorDir, ".argo")
		if err := copyContents(argoWorkflowsSource, argoWorkflowsDest, true); err != nil {
			return fmt.Errorf("error copying argo workflows content: %w", err)
		}

		// * copy $HOME/.k1/gitops/metaphor/Dockerfile $HOME/.k1/metaphor/build/Dockerfile
		dockerfileSource := filepath.Join(metaphorDir, "Dockerfile")
		dockerfileDest := filepath.Join(metaphorDir, "build", "Dockerfile")
		if err := copyContents(dockerfileSource, dockerfileDest, true); err != nil {
			return fmt.Errorf("error copying metaphor Dockerfile content: %w", err)
		}

		// Remove metaphor content from gitops repository directory
		if err := os.RemoveAll(metaphorLocation); err != nil {
			log.Error().Msgf("Error removing %q from filesystem: %s", metaphorLocation, err.Error())
			return fmt.Errorf("error removing %q from filesystem: %w", metaphorLocation, err)
		}
	}

	if err := gitClient.Commit(metaphorRepo, "init commit pre ref change"); err != nil {
		log.Error().Msgf("Error committing initial metaphor content: %s", err.Error())
		return fmt.Errorf("error committing initial metaphor content: %w", err)
	}

	metaphorRepo, err = gitClient.SetRefToMainBranch(metaphorRepo)
	if err != nil {
		log.Error().Msgf("Error setting ref to main branch: %s", err.Error())
		return fmt.Errorf("error setting ref to main branch: %w", err)
	}

	// remove old git ref
	err = metaphorRepo.Storer.RemoveReference(plumbing.NewBranchReferenceName("master"))
	if err != nil {
		log.Error().Msgf("Error removing previous git ref: %s", err.Error())
		return fmt.Errorf("error removing previous git ref: %w", err)
	}

	// create remote
	opts := &config.RemoteConfig{
		Name: "origin",
		URLs: []string{destinationMetaphorRepoURL},
	}

	if _, err = metaphorRepo.CreateRemote(opts); err != nil {
		log.Error().Msgf("Error creating remote for metaphor repository: %s", err.Error())
		return fmt.Errorf("error creating remote for metaphor repository: %w", err)
	}

	return nil
}

// PrepareGitRepositories
func PrepareGitRepositories(
	cloudProvider string,
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
	apexContentExists bool,
	gitProtocol string,
	useCloudflareOriginIssuer bool,
) error {
	// * clone the gitops-template repo
	gitopsRepo, err := gitClient.CloneRefSetMain(gitopsTemplateBranch, gitopsDir, gitopsTemplateURL)
	if err != nil {
		log.Error().Msgf("error opening repo at: %s, err: %s", gitopsDir, err.Error())
		return fmt.Errorf("error opening repo at: %s, err: %w", gitopsDir, err)
	}

	log.Info().Msg("gitops repository clone complete")

	// ADJUST CONTENT
	// * adjust the content for the gitops repo
	err = AdjustGitopsRepo(cloudProvider, clusterName, clusterType, gitopsDir, gitProvider, apexContentExists, useCloudflareOriginIssuer)
	if err != nil {
		log.Error().Msgf("unable to prepare repository: %s", err.Error())
		return fmt.Errorf("unable to prepare repository: %w", err)
	}

	// DETOKENIZE
	// * detokenize the gitops repo
	if err := DetokenizeGitGitops(gitopsDir, gitopsTokens, gitProtocol, useCloudflareOriginIssuer); err != nil {
		log.Error().Msgf("unable to detokenize gitops repository: %s", err.Error())
		return fmt.Errorf("unable to detokenize gitops repository: %w", err)
	}

	// ADJUST CONTENT
	// * adjust the content for the metaphor repo
	if err := AdjustMetaphorRepo(destinationMetaphorRepoURL, gitopsDir, gitProvider, cloudProvider, k1Dir); err != nil {
		log.Error().Msgf("unable to prepare metaphor repository: %s", err.Error())
		return fmt.Errorf("unable to prepare metaphor repository: %w", err)
	}

	// DETOKENIZE
	// * detokenize the metaphor repo
	if err := DetokenizeGitMetaphor(metaphorDir, metaphorTokens); err != nil {
		log.Error().Msgf("unable to detokenize metaphor repository: %s", err.Error())
		return fmt.Errorf("unable to detokenize metaphor repository: %w", err)
	}

	// COMMIT
	// * commit initial gitops-template content
	err = gitClient.Commit(gitopsRepo, "committing initial detokenized gitops-template repo content")
	if err != nil {
		log.Error().Msgf("unable to commit initial gitops-template content: %s", err.Error())
		return fmt.Errorf("unable to commit initial gitops-template content: %w", err)
	}

	// * commit initial metaphor content
	metaphorRepo, err := git.PlainOpen(metaphorDir)
	if err != nil {
		log.Error().Msgf("error opening metaphor git repository: %s", err.Error())
		return fmt.Errorf("error opening metaphor git repository: %w", err)
	}

	err = gitClient.Commit(metaphorRepo, "committing initial detokenized metaphor repo content")
	if err != nil {
		log.Error().Msgf("unable to commit initial metaphor content: %s", err.Error())
		return fmt.Errorf("unable to commit initial metaphor content: %w", err)
	}

	// ADD REMOTE(S)
	// * add new remote for gitops repo
	err = gitClient.AddRemote(destinationGitopsRepoURL, gitProvider, gitopsRepo)
	if err != nil {
		log.Error().Msgf("unable to add remote for gitops repo: %s", err.Error())
		return fmt.Errorf("unable to add remote for gitops repo: %w", err)
	}

	// * add new remote for metaphor repo
	err = gitClient.AddRemote(destinationMetaphorRepoURL, gitProvider, metaphorRepo)
	if err != nil {
		log.Error().Msgf("unable to add remote for metaphor repo: %s", err.Error())
		return fmt.Errorf("unable to add remote for metaphor repo: %w", err)
	}

	return nil
}
