/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package providerConfigs

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	pkg "github.com/kubefirst/kubefirst-api/internal"
	"github.com/kubefirst/kubefirst-api/internal/gitClient"

	cp "github.com/otiai10/copy"
	"github.com/rs/zerolog/log"
)

// AdjustGitopsRepo
func AdjustGitopsRepo(
	cloudProvider string,
	clusterName string,
	clusterType string,
	gitopsRepoDir string,
	gitProvider string,
	k1Dir string,
	apexContentExists bool,
	useCloudflareOriginIssuer bool,
) error {
	//* clean up all other platforms
	for _, platform := range pkg.SupportedPlatforms {
		if platform != fmt.Sprintf("%s-%s", cloudProvider, gitProvider) {
			os.RemoveAll(gitopsRepoDir + "/" + platform)
		}
	}

	//* copy options
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

	if !useCloudflareOriginIssuer {
		os.RemoveAll(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/mgmt/cloudflare-origin-ca-issuer.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
		os.RemoveAll(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/mgmt/cloudflare-origin-issuer-crd.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
		os.RemoveAll(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/mgmt/components/argo-workflows/cloudflareissuer.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
		os.RemoveAll(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/mgmt/components/argocd/cloudflareissuer.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
		os.RemoveAll(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/mgmt/components/atlantis/cloudflareissuer.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
		os.RemoveAll(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/mgmt/components/chartmuseum/cloudflareissuer.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
		os.RemoveAll(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/mgmt/components/kubefirst/cloudflareissuer.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
		os.RemoveAll(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/mgmt/components/vault/cloudflareissuer.yaml", gitopsRepoDir, cloudProvider, gitProvider)))

		os.RemoveAll(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/workload-cluster/cloudflare-origin-issuer", gitopsRepoDir, cloudProvider, gitProvider)))
		os.RemoveAll(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/workload-cluster/40-cloudflare-origin-issuer-crd.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
		os.RemoveAll(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/workload-cluster/41-cloudflare-origin-ca-issuer.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
		os.RemoveAll(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/workload-cluster/45-cloudflare-origin-issuer.yaml", gitopsRepoDir, cloudProvider, gitProvider)))

		os.RemoveAll(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/workload-vcluster/cloudflare-origin-issuer", gitopsRepoDir, cloudProvider, gitProvider)))
		os.RemoveAll(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/workload-vcluster/40-cloudflare-origin-issuer-crd.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
		os.RemoveAll(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/workload-vcluster/41-cloudflare-origin-ca-issuer.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
		os.RemoveAll(strings.ToLower(fmt.Sprintf("%s/%s-%s/templates/workload-vcluster/45-cloudflare-origin-issuer.yaml", gitopsRepoDir, cloudProvider, gitProvider)))
	}

	AKAMAI_GITHUB := "akamai-github" //! i know i know i know.

	if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == AKAMAI_GITHUB {
		driverContent := fmt.Sprintf("%s/%s-%s/", gitopsRepoDir, cloudProvider, gitProvider)
		err := cp.Copy(driverContent, gitopsRepoDir, opt)
		if err != nil {
			log.Info().Msgf("Error populating gitops repository with driver content: %s. error: %s", fmt.Sprintf("%s-%s", cloudProvider, gitProvider), err.Error())
			return err
		}
		os.RemoveAll(driverContent)

		//* copy $HOME/.k1/gitops/templates/${clusterType}/* $HOME/.k1/gitops/registry/${clusterName}
		clusterContent := fmt.Sprintf("%s/templates/%s", gitopsRepoDir, clusterType)

		// Remove apex content if apex content already exists
		if apexContentExists {
			log.Warn().Msgf("removing nginx-apex since apexContentExists was %v", apexContentExists)
			os.Remove(fmt.Sprintf("%s/nginx-apex.yaml", clusterContent))
			os.RemoveAll(fmt.Sprintf("%s/nginx-apex", clusterContent))
		} else {
			log.Warn().Msgf("will create nginx-apex since apexContentExists was %v", apexContentExists)
		}

		if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == AKAMAI_GITHUB {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/clusters/%s", gitopsRepoDir, clusterName), opt)
		} else {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/%s", gitopsRepoDir, clusterName), opt)
		}
		if err != nil {
			log.Info().Msgf("Error populating cluster content with %s. error: %s", clusterContent, err.Error())
			return err
		}
		os.RemoveAll(fmt.Sprintf("%s/templates/mgmt", gitopsRepoDir))

		return nil
	}

	AWS_GITHUB := "aws-github"

	if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == AWS_GITHUB {
		driverContent := fmt.Sprintf("%s/%s-%s/", gitopsRepoDir, cloudProvider, gitProvider)
		err := cp.Copy(driverContent, gitopsRepoDir, opt)
		if err != nil {
			log.Info().Msgf("Error populating gitops repository with driver content: %s. error: %s", fmt.Sprintf("%s-%s", cloudProvider, gitProvider), err.Error())
			return err
		}
		os.RemoveAll(driverContent)

		//* copy $HOME/.k1/gitops/templates/${clusterType}/* $HOME/.k1/gitops/registry/${clusterName}
		clusterContent := fmt.Sprintf("%s/templates/%s", gitopsRepoDir, clusterType)

		// Remove apex content if apex content already exists
		if apexContentExists {
			log.Warn().Msgf("removing nginx-apex since apexContentExists was %v", apexContentExists)
			os.Remove(fmt.Sprintf("%s/nginx-apex.yaml", clusterContent))
			os.RemoveAll(fmt.Sprintf("%s/nginx-apex", clusterContent))
		} else {
			log.Warn().Msgf("will create nginx-apex since apexContentExists was %v", apexContentExists)
		}

		if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == AWS_GITHUB {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/clusters/%s", gitopsRepoDir, clusterName), opt)
		} else {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/%s", gitopsRepoDir, clusterName), opt)
		}
		if err != nil {
			log.Info().Msgf("Error populating cluster content with %s. error: %s", clusterContent, err.Error())
			return err
		}
		os.RemoveAll(fmt.Sprintf("%s/templates/mgmt", gitopsRepoDir))

		return nil
	}

	AWS_GITLAB := "aws-gitlab"

	if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == AWS_GITLAB {
		driverContent := fmt.Sprintf("%s/%s-%s/", gitopsRepoDir, cloudProvider, gitProvider)
		err := cp.Copy(driverContent, gitopsRepoDir, opt)
		if err != nil {
			log.Info().Msgf("Error populating gitops repository with driver content: %s. error: %s", fmt.Sprintf("%s-%s", cloudProvider, gitProvider), err.Error())
			return err
		}
		os.RemoveAll(driverContent)

		//* copy $HOME/.k1/gitops/templates/${clusterType}/* $HOME/.k1/gitops/registry/${clusterName}
		clusterContent := fmt.Sprintf("%s/templates/%s", gitopsRepoDir, clusterType)

		// Remove apex content if apex content already exists
		if apexContentExists {
			log.Warn().Msgf("removing nginx-apex since apexContentExists was %v", apexContentExists)
			os.Remove(fmt.Sprintf("%s/nginx-apex.yaml", clusterContent))
			os.RemoveAll(fmt.Sprintf("%s/nginx-apex", clusterContent))
		} else {
			log.Warn().Msgf("will create nginx-apex since apexContentExists was %v", apexContentExists)
		}

		if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == AWS_GITLAB {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/clusters/%s", gitopsRepoDir, clusterName), opt)
		} else {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/%s", gitopsRepoDir, clusterName), opt)
		}
		if err != nil {
			log.Info().Msgf("Error populating cluster content with %s. error: %s", clusterContent, err.Error())
			return err
		}
		os.RemoveAll(fmt.Sprintf("%s/templates/mgmt", gitopsRepoDir))

		return nil
	}

	CIVO_GITHUB := "civo-github" //! i know i know i know.

	if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == CIVO_GITHUB {
		driverContent := fmt.Sprintf("%s/%s-%s/", gitopsRepoDir, cloudProvider, gitProvider)
		err := cp.Copy(driverContent, gitopsRepoDir, opt)
		if err != nil {
			log.Info().Msgf("Error populating gitops repository with driver content: %s. error: %s", fmt.Sprintf("%s-%s", cloudProvider, gitProvider), err.Error())
			return err
		}
		os.RemoveAll(driverContent)

		//* copy $HOME/.k1/gitops/templates/${clusterType}/* $HOME/.k1/gitops/registry/${clusterName}
		clusterContent := fmt.Sprintf("%s/templates/%s", gitopsRepoDir, clusterType)

		// Remove apex content if apex content already exists
		if apexContentExists {
			log.Warn().Msgf("removing nginx-apex since apexContentExists was %v", apexContentExists)
			os.Remove(fmt.Sprintf("%s/nginx-apex.yaml", clusterContent))
			os.RemoveAll(fmt.Sprintf("%s/nginx-apex", clusterContent))
		} else {
			log.Warn().Msgf("will create nginx-apex since apexContentExists was %v", apexContentExists)
		}

		if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == CIVO_GITHUB {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/clusters/%s", gitopsRepoDir, clusterName), opt)
		} else {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/%s", gitopsRepoDir, clusterName), opt)
		}
		if err != nil {
			log.Info().Msgf("Error populating cluster content with %s. error: %s", clusterContent, err.Error())
			return err
		}
		os.RemoveAll(fmt.Sprintf("%s/templates/mgmt", gitopsRepoDir))

		return nil
	}

	CIVO_GITLAB := "civo-gitlab"

	if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == CIVO_GITLAB {
		driverContent := fmt.Sprintf("%s/%s-%s/", gitopsRepoDir, cloudProvider, gitProvider)
		err := cp.Copy(driverContent, gitopsRepoDir, opt)
		if err != nil {
			log.Info().Msgf("Error populating gitops repository with driver content: %s. error: %s", fmt.Sprintf("%s-%s", cloudProvider, gitProvider), err.Error())
			return err
		}
		os.RemoveAll(driverContent)

		//* copy $HOME/.k1/gitops/templates/${clusterType}/* $HOME/.k1/gitops/registry/${clusterName}
		clusterContent := fmt.Sprintf("%s/templates/%s", gitopsRepoDir, clusterType)

		// Remove apex content if apex content already exists
		if apexContentExists {
			log.Warn().Msgf("removing nginx-apex since apexContentExists was %v", apexContentExists)
			os.Remove(fmt.Sprintf("%s/nginx-apex.yaml", clusterContent))
			os.RemoveAll(fmt.Sprintf("%s/nginx-apex", clusterContent))
		} else {
			log.Warn().Msgf("will create nginx-apex since apexContentExists was %v", apexContentExists)
		}

		if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == CIVO_GITLAB {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/clusters/%s", gitopsRepoDir, clusterName), opt)
		} else {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/%s", gitopsRepoDir, clusterName), opt)
		}
		if err != nil {
			log.Info().Msgf("Error populating cluster content with %s. error: %s", clusterContent, err.Error())
			return err
		}
		os.RemoveAll(fmt.Sprintf("%s/templates/mgmt", gitopsRepoDir))

		return nil
	}
	GOOGLE_GITHUB := "google-github"

	if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == GOOGLE_GITHUB {
		driverContent := fmt.Sprintf("%s/%s-%s/", gitopsRepoDir, cloudProvider, gitProvider)
		err := cp.Copy(driverContent, gitopsRepoDir, opt)
		if err != nil {
			log.Info().Msgf("Error populating gitops repository with driver content: %s. error: %s", fmt.Sprintf("%s-%s", cloudProvider, gitProvider), err.Error())
			return err
		}
		os.RemoveAll(driverContent)

		//* copy $HOME/.k1/gitops/templates/${clusterType}/* $HOME/.k1/gitops/registry/${clusterName}
		clusterContent := fmt.Sprintf("%s/templates/%s", gitopsRepoDir, clusterType)

		// Remove apex content if apex content already exists
		if apexContentExists {
			log.Warn().Msgf("removing nginx-apex since apexContentExists was %v", apexContentExists)
			os.Remove(fmt.Sprintf("%s/nginx-apex.yaml", clusterContent))
			os.RemoveAll(fmt.Sprintf("%s/nginx-apex", clusterContent))
		} else {
			log.Warn().Msgf("will create nginx-apex since apexContentExists was %v", apexContentExists)
		}

		if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == GOOGLE_GITHUB {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/clusters/%s", gitopsRepoDir, clusterName), opt)
		} else {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/%s", gitopsRepoDir, clusterName), opt)
		}
		if err != nil {
			log.Info().Msgf("Error populating cluster content with %s. error: %s", clusterContent, err.Error())
			return err
		}
		os.RemoveAll(fmt.Sprintf("%s/templates/mgmt", gitopsRepoDir))

		return nil
	}

	GOOGLE_GITLAB := "google-gitlab"

	if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == GOOGLE_GITLAB {
		driverContent := fmt.Sprintf("%s/%s-%s/", gitopsRepoDir, cloudProvider, gitProvider)
		err := cp.Copy(driverContent, gitopsRepoDir, opt)
		if err != nil {
			log.Info().Msgf("Error populating gitops repository with driver content: %s. error: %s", fmt.Sprintf("%s-%s", cloudProvider, gitProvider), err.Error())
			return err
		}
		os.RemoveAll(driverContent)

		//* copy $HOME/.k1/gitops/templates/${clusterType}/* $HOME/.k1/gitops/registry/${clusterName}
		clusterContent := fmt.Sprintf("%s/templates/%s", gitopsRepoDir, clusterType)

		// Remove apex content if apex content already exists
		if apexContentExists {
			log.Warn().Msgf("removing nginx-apex since apexContentExists was %v", apexContentExists)
			os.Remove(fmt.Sprintf("%s/nginx-apex.yaml", clusterContent))
			os.RemoveAll(fmt.Sprintf("%s/nginx-apex", clusterContent))
		} else {
			log.Warn().Msgf("will create nginx-apex since apexContentExists was %v", apexContentExists)
		}

		if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == GOOGLE_GITLAB {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/clusters/%s", gitopsRepoDir, clusterName), opt)
		} else {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/%s", gitopsRepoDir, clusterName), opt)
		}
		if err != nil {
			log.Info().Msgf("Error populating cluster content with %s. error: %s", clusterContent, err.Error())
			return err
		}
		os.RemoveAll(fmt.Sprintf("%s/templates/mgmt", gitopsRepoDir))

		return nil
	}

	DIGITALOCEAN_GITHUB := "digitalocean-github"

	if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == DIGITALOCEAN_GITHUB {
		driverContent := fmt.Sprintf("%s/%s-%s/", gitopsRepoDir, cloudProvider, gitProvider)
		err := cp.Copy(driverContent, gitopsRepoDir, opt)
		if err != nil {
			log.Info().Msgf("Error populating gitops repository with driver content: %s. error: %s", fmt.Sprintf("%s-%s", cloudProvider, gitProvider), err.Error())
			return err
		}
		os.RemoveAll(driverContent)

		//* copy $HOME/.k1/gitops/templates/${clusterType}/* $HOME/.k1/gitops/registry/${clusterName}
		clusterContent := fmt.Sprintf("%s/templates/%s", gitopsRepoDir, clusterType)

		// Remove apex content if apex content already exists
		if apexContentExists {
			log.Warn().Msgf("removing nginx-apex since apexContentExists was %v", apexContentExists)
			os.Remove(fmt.Sprintf("%s/nginx-apex.yaml", clusterContent))
			os.RemoveAll(fmt.Sprintf("%s/nginx-apex", clusterContent))
		} else {
			log.Warn().Msgf("will create nginx-apex since apexContentExists was %v", apexContentExists)
		}

		if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == DIGITALOCEAN_GITHUB {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/clusters/%s", gitopsRepoDir, clusterName), opt)
		} else {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/%s", gitopsRepoDir, clusterName), opt)
		}
		if err != nil {
			log.Info().Msgf("Error populating cluster content with %s. error: %s", clusterContent, err.Error())
			return err
		}
		os.RemoveAll(fmt.Sprintf("%s/templates/mgmt", gitopsRepoDir))

		return nil
	}

	DIGITALOCEAN_GITLAB := "digitalocean-gitlab"

	if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == DIGITALOCEAN_GITLAB {
		driverContent := fmt.Sprintf("%s/%s-%s/", gitopsRepoDir, cloudProvider, gitProvider)
		err := cp.Copy(driverContent, gitopsRepoDir, opt)
		if err != nil {
			log.Info().Msgf("Error populating gitops repository with driver content: %s. error: %s", fmt.Sprintf("%s-%s", cloudProvider, gitProvider), err.Error())
			return err
		}
		os.RemoveAll(driverContent)

		//* copy $HOME/.k1/gitops/templates/${clusterType}/* $HOME/.k1/gitops/registry/${clusterName}
		clusterContent := fmt.Sprintf("%s/templates/%s", gitopsRepoDir, clusterType)

		// Remove apex content if apex content already exists
		if apexContentExists {
			log.Warn().Msgf("removing nginx-apex since apexContentExists was %v", apexContentExists)
			os.Remove(fmt.Sprintf("%s/nginx-apex.yaml", clusterContent))
			os.RemoveAll(fmt.Sprintf("%s/nginx-apex", clusterContent))
		} else {
			log.Warn().Msgf("will create nginx-apex since apexContentExists was %v", apexContentExists)
		}

		if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == DIGITALOCEAN_GITLAB {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/clusters/%s", gitopsRepoDir, clusterName), opt)
		} else {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/%s", gitopsRepoDir, clusterName), opt)
		}
		if err != nil {
			log.Info().Msgf("Error populating cluster content with %s. error: %s", clusterContent, err.Error())
			return err
		}
		os.RemoveAll(fmt.Sprintf("%s/templates/mgmt", gitopsRepoDir))

		return nil
	}

	VULTR_GITHUB := "vultr-github"

	if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == VULTR_GITHUB {
		driverContent := fmt.Sprintf("%s/%s-%s/", gitopsRepoDir, cloudProvider, gitProvider)
		err := cp.Copy(driverContent, gitopsRepoDir, opt)
		if err != nil {
			log.Info().Msgf("Error populating gitops repository with driver content: %s. error: %s", fmt.Sprintf("%s-%s", cloudProvider, gitProvider), err.Error())
			return err
		}
		os.RemoveAll(driverContent)

		//* copy $HOME/.k1/gitops/templates/${clusterType}/* $HOME/.k1/gitops/registry/${clusterName}
		clusterContent := fmt.Sprintf("%s/templates/%s", gitopsRepoDir, clusterType)

		// Remove apex content if apex content already exists
		if apexContentExists {
			log.Warn().Msgf("removing nginx-apex since apexContentExists was %v", apexContentExists)
			os.Remove(fmt.Sprintf("%s/nginx-apex.yaml", clusterContent))
			os.RemoveAll(fmt.Sprintf("%s/nginx-apex", clusterContent))
		} else {
			log.Warn().Msgf("will create nginx-apex since apexContentExists was %v", apexContentExists)
		}

		if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == VULTR_GITHUB {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/clusters/%s", gitopsRepoDir, clusterName), opt)
		} else {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/%s", gitopsRepoDir, clusterName), opt)
		}
		if err != nil {
			log.Info().Msgf("Error populating cluster content with %s. error: %s", clusterContent, err.Error())
			return err
		}
		os.RemoveAll(fmt.Sprintf("%s/templates/mgmt", gitopsRepoDir))

		return nil
	}

	VULTR_GITLAB := "vultr-gitlab"

	if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == VULTR_GITLAB {
		driverContent := fmt.Sprintf("%s/%s-%s/", gitopsRepoDir, cloudProvider, gitProvider)
		err := cp.Copy(driverContent, gitopsRepoDir, opt)
		if err != nil {
			log.Info().Msgf("Error populating gitops repository with driver content: %s. error: %s", fmt.Sprintf("%s-%s", cloudProvider, gitProvider), err.Error())
			return err
		}
		os.RemoveAll(driverContent)

		//* copy $HOME/.k1/gitops/templates/${clusterType}/* $HOME/.k1/gitops/registry/${clusterName}
		clusterContent := fmt.Sprintf("%s/templates/%s", gitopsRepoDir, clusterType)

		// Remove apex content if apex content already exists
		if apexContentExists {
			log.Warn().Msgf("removing nginx-apex since apexContentExists was %v", apexContentExists)
			os.Remove(fmt.Sprintf("%s/nginx-apex.yaml", clusterContent))
			os.RemoveAll(fmt.Sprintf("%s/nginx-apex", clusterContent))
		} else {
			log.Warn().Msgf("will create nginx-apex since apexContentExists was %v", apexContentExists)
		}

		if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == VULTR_GITLAB {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/clusters/%s", gitopsRepoDir, clusterName), opt)
		} else {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/%s", gitopsRepoDir, clusterName), opt)
		}
		if err != nil {
			log.Info().Msgf("Error populating cluster content with %s. error: %s", clusterContent, err.Error())
			return err
		}
		os.RemoveAll(fmt.Sprintf("%s/templates/mgmt", gitopsRepoDir))

		return nil
	}

	K3S_GITLAB := "k3s-gitlab"

	if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == K3S_GITLAB {
		driverContent := fmt.Sprintf("%s/%s-%s/", gitopsRepoDir, cloudProvider, gitProvider)
		err := cp.Copy(driverContent, gitopsRepoDir, opt)
		if err != nil {
			log.Info().Msgf("Error populating gitops repository with driver content: %s. error: %s", fmt.Sprintf("%s-%s", cloudProvider, gitProvider), err.Error())
			return err
		}
		os.RemoveAll(driverContent)

		//* copy $HOME/.k1/gitops/templates/${clusterType}/* $HOME/.k1/gitops/registry/${clusterName}
		clusterContent := fmt.Sprintf("%s/templates/%s", gitopsRepoDir, clusterType)

		// Remove apex content if apex content already exists
		if apexContentExists {
			log.Warn().Msgf("removing nginx-apex since apexContentExists was %v", apexContentExists)
			os.Remove(fmt.Sprintf("%s/nginx-apex.yaml", clusterContent))
			os.RemoveAll(fmt.Sprintf("%s/nginx-apex", clusterContent))
		} else {
			log.Warn().Msgf("will create nginx-apex since apexContentExists was %v", apexContentExists)
		}

		if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == K3S_GITLAB {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/clusters/%s", gitopsRepoDir, clusterName), opt)
		} else {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/%s", gitopsRepoDir, clusterName), opt)
		}
		if err != nil {
			log.Info().Msgf("Error populating cluster content with %s. error: %s", clusterContent, err.Error())
			return err
		}
		os.RemoveAll(fmt.Sprintf("%s/templates/mgmt", gitopsRepoDir))

		return nil
	}

	K3S_GITHUB := "k3s-github"

	if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == K3S_GITHUB {
		driverContent := fmt.Sprintf("%s/%s-%s/", gitopsRepoDir, cloudProvider, gitProvider)
		err := cp.Copy(driverContent, gitopsRepoDir, opt)
		if err != nil {
			log.Info().Msgf("Error populating gitops repository with driver content: %s. error: %s", fmt.Sprintf("%s-%s", cloudProvider, gitProvider), err.Error())
			return err
		}
		os.RemoveAll(driverContent)

		//* copy $HOME/.k1/gitops/templates/${clusterType}/* $HOME/.k1/gitops/registry/${clusterName}
		clusterContent := fmt.Sprintf("%s/templates/%s", gitopsRepoDir, clusterType)

		// Remove apex content if apex content already exists
		if apexContentExists {
			log.Warn().Msgf("removing nginx-apex since apexContentExists was %v", apexContentExists)
			os.Remove(fmt.Sprintf("%s/nginx-apex.yaml", clusterContent))
			os.RemoveAll(fmt.Sprintf("%s/nginx-apex", clusterContent))
		} else {
			log.Warn().Msgf("will create nginx-apex since apexContentExists was %v", apexContentExists)
		}

		if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == K3S_GITHUB {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/clusters/%s", gitopsRepoDir, clusterName), opt)
		} else {
			err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/%s", gitopsRepoDir, clusterName), opt)
		}
		if err != nil {
			log.Info().Msgf("Error populating cluster content with %s. error: %s", clusterContent, err.Error())
			return err
		}
		os.RemoveAll(fmt.Sprintf("%s/templates/mgmt", gitopsRepoDir))

		return nil
	}

	//* copy $cloudProvider-$gitProvider/* $HOME/.k1/gitops/
	driverContent := fmt.Sprintf("%s/%s-%s/", gitopsRepoDir, cloudProvider, gitProvider)
	err := cp.Copy(driverContent, gitopsRepoDir, opt)
	if err != nil {
		log.Info().Msgf("Error populating gitops repository with driver content: %s. error: %s", fmt.Sprintf("%s-%s", cloudProvider, gitProvider), err.Error())
		return err
	}
	os.RemoveAll(driverContent)

	//* copy $HOME/.k1/gitops/cluster-types/${clusterType}/* $HOME/.k1/gitops/registry/${clusterName}
	clusterContent := fmt.Sprintf("%s/cluster-types/%s", gitopsRepoDir, clusterType)

	// Remove apex content if apex content already exists
	if apexContentExists {
		log.Warn().Msgf("removing nginx-apex since apexContentExists was %v", apexContentExists)
		os.Remove(fmt.Sprintf("%s/nginx-apex.yaml", clusterContent))
		os.RemoveAll(fmt.Sprintf("%s/nginx-apex", clusterContent))
	} else {
		log.Warn().Msgf("will create nginx-apex since apexContentExists was %v", apexContentExists)
	}

	if strings.ToLower(fmt.Sprintf("%s-%s", cloudProvider, gitProvider)) == CIVO_GITHUB {
		err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/clusters/%s", gitopsRepoDir, clusterName), opt)
	} else {
		err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/%s", gitopsRepoDir, clusterName), opt)
	}
	if err != nil {
		log.Info().Msgf("Error populating cluster content with %s. error: %s", clusterContent, err.Error())
		return err
	}
	os.RemoveAll(fmt.Sprintf("%s/cluster-types", gitopsRepoDir))
	os.RemoveAll(fmt.Sprintf("%s/services", gitopsRepoDir))

	return nil
}

// AdjustMetaphorRepo
func AdjustMetaphorRepo(
	destinationMetaphorRepoURL string,
	gitopsRepoDir string,
	gitProvider string,
	metaphorRepoName,
	k1Dir string,
) error {
	//* create ~/.k1/metaphor
	metaphorDir := fmt.Sprintf("%s/%s", k1Dir, metaphorRepoName)
	os.Mkdir(metaphorDir, 0700)

	//* git init
	metaphorRepo, err := git.PlainInit(metaphorDir, false)
	if err != nil {
		return err
	}

	//* copy options
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

	AKAMAI_GITHUB := "akamai-github"

	if strings.ToLower(fmt.Sprintf("akamai-%s", gitProvider)) != AKAMAI_GITHUB {
		os.RemoveAll(metaphorDir + "/.argo")
		os.RemoveAll(metaphorDir + "/.github")
	}

	//todo implement repo, err :- createMetaphor() which returns the metaphor repoository object, removes content from
	// gitops and then allows gitops to commit during its sequence of ops
	if strings.ToLower(fmt.Sprintf("akamai-%s", gitProvider)) == AKAMAI_GITHUB {
		//* metaphor app source
		metaphorContent := fmt.Sprintf("%s/metaphor", gitopsRepoDir)
		err = cp.Copy(metaphorContent, metaphorDir, opt)
		if err != nil {
			log.Info().Msgf("Error populating metaphor content with %s. error: %s", metaphorContent, err.Error())
			return err
		}

		// Remove metaphor content from gitops repository directory
		os.RemoveAll(fmt.Sprintf("%s/metaphor", gitopsRepoDir))

		err = gitClient.Commit(metaphorRepo, "init commit pre ref change")
		if err != nil {
			return err
		}

		metaphorRepo, err = gitClient.SetRefToMainBranch(metaphorRepo)
		if err != nil {
			return err
		}

		// remove old git ref
		err = metaphorRepo.Storer.RemoveReference(plumbing.NewBranchReferenceName("master"))
		if err != nil {
			return fmt.Errorf("error removing previous git ref: %s", err)
		}

		// create remote
		_, err = metaphorRepo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{destinationMetaphorRepoURL},
		})
		if err != nil {
			return fmt.Errorf("error creating remote for metaphor repository: %s", err)
		}

		return nil

	}

	AWS_GITHUB := "aws-github"

	if strings.ToLower(fmt.Sprintf("aws-%s", gitProvider)) != AWS_GITHUB {
		os.RemoveAll(metaphorDir + "/.argo")
		os.RemoveAll(metaphorDir + "/.github")
	}

	// todo implement repo, err :- createMetaphor() which returns the metaphor repoository object, removes content from
	// gitops and then allows gitops to commit during its sequence of ops
	if strings.ToLower(fmt.Sprintf("aws-%s", gitProvider)) == AWS_GITHUB {
		//* metaphor app source
		metaphorContent := fmt.Sprintf("%s/metaphor", gitopsRepoDir)
		err = cp.Copy(metaphorContent, metaphorDir, opt)
		if err != nil {
			log.Info().Msgf("Error populating metaphor content with %s. error: %s", metaphorContent, err.Error())
			return err
		}

		// Remove metaphor content from gitops repository directory
		os.RemoveAll(fmt.Sprintf("%s/metaphor", gitopsRepoDir))

		err = gitClient.Commit(metaphorRepo, "init commit pre ref change")
		if err != nil {
			return err
		}

		metaphorRepo, err = gitClient.SetRefToMainBranch(metaphorRepo)
		if err != nil {
			return err
		}

		// remove old git ref
		err = metaphorRepo.Storer.RemoveReference(plumbing.NewBranchReferenceName("master"))
		if err != nil {
			return fmt.Errorf("error removing previous git ref: %s", err)
		}

		// create remote
		_, err = metaphorRepo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{destinationMetaphorRepoURL},
		})
		if err != nil {
			return fmt.Errorf("error creating remote for metaphor repository: %s", err)
		}

		return nil

	}

	AWS_GITLAB := "aws-gitlab"

	if strings.ToLower(fmt.Sprintf("aws-%s", gitProvider)) != AWS_GITLAB {
		os.RemoveAll(metaphorDir + "/.argo")
		os.RemoveAll(metaphorDir + "/.github")
	}

	// todo implement repo, err :- createMetaphor() which returns the metaphor repoository object, removes content from
	// gitops and then allows gitops to commit during its sequence of ops
	if strings.ToLower(fmt.Sprintf("aws-%s", gitProvider)) == AWS_GITLAB {
		//* metaphor app source
		metaphorContent := fmt.Sprintf("%s/metaphor", gitopsRepoDir)
		err = cp.Copy(metaphorContent, metaphorDir, opt)
		if err != nil {
			log.Info().Msgf("Error populating metaphor content with %s. error: %s", metaphorContent, err.Error())
			return err
		}

		// Remove metaphor content from gitops repository directory
		os.RemoveAll(fmt.Sprintf("%s/metaphor", gitopsRepoDir))

		err = gitClient.Commit(metaphorRepo, "init commit pre ref change")
		if err != nil {
			return err
		}

		metaphorRepo, err = gitClient.SetRefToMainBranch(metaphorRepo)
		if err != nil {
			return err
		}

		// remove old git ref
		err = metaphorRepo.Storer.RemoveReference(plumbing.NewBranchReferenceName("master"))
		if err != nil {
			return fmt.Errorf("error removing previous git ref: %s", err)
		}

		// create remote
		_, err = metaphorRepo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{destinationMetaphorRepoURL},
		})
		if err != nil {
			return fmt.Errorf("error creating remote for metaphor repository: %s", err)
		}

		return nil

	}

	CIVO_GITHUB := "civo-github"

	if strings.ToLower(fmt.Sprintf("civo-%s", gitProvider)) != CIVO_GITHUB {
		os.RemoveAll(metaphorDir + "/.argo")
		os.RemoveAll(metaphorDir + "/.github")
	}

	// todo implement repo, err :- createMetaphor() which returns the metaphor repoository object, removes content from
	// gitops and then allows gitops to commit during its sequence of ops
	if strings.ToLower(fmt.Sprintf("civo-%s", gitProvider)) == CIVO_GITHUB {
		//* metaphor app source
		metaphorContent := fmt.Sprintf("%s/metaphor", gitopsRepoDir)
		err = cp.Copy(metaphorContent, metaphorDir, opt)
		if err != nil {
			log.Info().Msgf("Error populating metaphor content with %s. error: %s", metaphorContent, err.Error())
			return err
		}

		// Remove metaphor content from gitops repository directory
		os.RemoveAll(fmt.Sprintf("%s/metaphor", gitopsRepoDir))

		err = gitClient.Commit(metaphorRepo, "init commit pre ref change")
		if err != nil {
			return err
		}

		metaphorRepo, err = gitClient.SetRefToMainBranch(metaphorRepo)
		if err != nil {
			return err
		}

		// remove old git ref
		err = metaphorRepo.Storer.RemoveReference(plumbing.NewBranchReferenceName("master"))
		if err != nil {
			return fmt.Errorf("error removing previous git ref: %s", err)
		}

		// create remote
		_, err = metaphorRepo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{destinationMetaphorRepoURL},
		})
		if err != nil {
			return fmt.Errorf("error creating remote for metaphor repository: %s", err)
		}

		return nil

	}

	CIVO_GITLAB := "civo-gitlab"

	if strings.ToLower(fmt.Sprintf("civo-%s", gitProvider)) != CIVO_GITLAB {
		os.RemoveAll(metaphorDir + "/.argo")
		os.RemoveAll(metaphorDir + "/.github")
	}

	// todo implement repo, err :- createMetaphor() which returns the metaphor repoository object, removes content from
	// gitops and then allows gitops to commit during its sequence of ops
	if strings.ToLower(fmt.Sprintf("civo-%s", gitProvider)) == CIVO_GITLAB {
		//* metaphor app source
		metaphorContent := fmt.Sprintf("%s/metaphor", gitopsRepoDir)
		err = cp.Copy(metaphorContent, metaphorDir, opt)
		if err != nil {
			log.Info().Msgf("Error populating metaphor content with %s. error: %s", metaphorContent, err.Error())
			return err
		}

		// Remove metaphor content from gitops repository directory
		os.RemoveAll(fmt.Sprintf("%s/metaphor", gitopsRepoDir))

		err = gitClient.Commit(metaphorRepo, "init commit pre ref change")
		if err != nil {
			return err
		}

		metaphorRepo, err = gitClient.SetRefToMainBranch(metaphorRepo)
		if err != nil {
			return err
		}

		// remove old git ref
		err = metaphorRepo.Storer.RemoveReference(plumbing.NewBranchReferenceName("master"))
		if err != nil {
			return fmt.Errorf("error removing previous git ref: %s", err)
		}

		// create remote
		_, err = metaphorRepo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{destinationMetaphorRepoURL},
		})
		if err != nil {
			return fmt.Errorf("error creating remote for metaphor repository: %s", err)
		}

		return nil

	}

	//* copy ci content
	switch gitProvider {
	case "github":
		//* copy $HOME/.k1/gitops/ci/.github/* $HOME/.k1/metaphor/.github
		githubActionsFolderContent := fmt.Sprintf("%s/gitops/ci/.github", k1Dir)
		log.Info().Msgf("copying github content: %s", githubActionsFolderContent)
		err := cp.Copy(githubActionsFolderContent, fmt.Sprintf("%s/.github", metaphorDir), opt)
		if err != nil {
			log.Info().Msgf("error populating metaphor repository with %s: %s", githubActionsFolderContent, err)
			return err
		}
	case "gitlab":
		//* copy $HOME/.k1/gitops/ci/.gitlab-ci.yml/* $HOME/.k1/metaphor/.github
		gitlabCIContent := fmt.Sprintf("%s/gitops/ci/.gitlab-ci.yml", k1Dir)
		log.Info().Msgf("copying gitlab content: %s", gitlabCIContent)
		err := cp.Copy(gitlabCIContent, fmt.Sprintf("%s/.gitlab-ci.yml", metaphorDir), opt)
		if err != nil {
			log.Info().Msgf("error populating metaphor repository with %s: %s", gitlabCIContent, err)
			return err
		}
	}

	//* metaphor app source
	metaphorContent := fmt.Sprintf("%s/metaphor", gitopsRepoDir)
	err = cp.Copy(metaphorContent, metaphorDir, opt)
	if err != nil {
		log.Info().Msgf("Error populating metaphor content with %s. error: %s", metaphorContent, err.Error())
		return err
	}

	//* copy $HOME/.k1/gitops/ci/.argo/* $HOME/.k1/metaphor/.argo
	argoWorkflowsFolderContent := fmt.Sprintf("%s/gitops/ci/.argo", k1Dir)
	log.Info().Msgf("copying argo workflows content: %s", argoWorkflowsFolderContent)
	err = cp.Copy(argoWorkflowsFolderContent, fmt.Sprintf("%s/.argo", metaphorDir), opt)
	if err != nil {
		log.Info().Msgf("error populating metaphor repository with %s: %s", argoWorkflowsFolderContent, err)
		return err
	}

	//* copy $HOME/.k1/gitops/metaphor/Dockerfile $HOME/.k1/metaphor/build/Dockerfile
	dockerfileContent := fmt.Sprintf("%s/Dockerfile", metaphorDir)
	os.Mkdir(metaphorDir+"/build", 0700)
	log.Info().Msgf("copying dockerfile content: %s", argoWorkflowsFolderContent)
	err = cp.Copy(dockerfileContent, fmt.Sprintf("%s/build/Dockerfile", metaphorDir), opt)
	if err != nil {
		log.Info().Msgf("error populating metaphor repository with %s: %s", argoWorkflowsFolderContent, err)
		return err
	}

	// Remove metaphor content from gitops repository directory
	os.RemoveAll(fmt.Sprintf("%s/metaphor", gitopsRepoDir))

	err = gitClient.Commit(metaphorRepo, "init commit pre ref change")
	if err != nil {
		return err
	}

	metaphorRepo, err = gitClient.SetRefToMainBranch(metaphorRepo)
	if err != nil {
		return err
	}

	// remove old git ref
	err = metaphorRepo.Storer.RemoveReference(plumbing.NewBranchReferenceName("master"))
	if err != nil {
		return fmt.Errorf("error removing previous git ref: %s", err)
	}

	// create remote
	_, err = metaphorRepo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{destinationMetaphorRepoURL},
	})
	if err != nil {
		return fmt.Errorf("error creating remote for metaphor repository: %s", err)
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
	//* clone the gitops-template repo
	gitopsRepo, err := gitClient.CloneRefSetMain(gitopsTemplateBranch, gitopsDir, gitopsTemplateURL)
	if err != nil {
		log.Panic().Msgf("error opening repo at: %s, err: %v", gitopsDir, err)
	}
	log.Info().Msg("gitops repository clone complete")

	// ADJUST CONTENT
	//* adjust the content for the gitops repo
	err = AdjustGitopsRepo(cloudProvider, clusterName, clusterType, gitopsDir, gitProvider, k1Dir, apexContentExists, useCloudflareOriginIssuer)
	if err != nil {
		log.Info().Msgf("err: %v", err)
		return err
	}

	// DETOKENIZE
	//* detokenize the gitops repo
	DetokenizeGitGitops(gitopsDir, gitopsTokens, gitProtocol, useCloudflareOriginIssuer)
	if err != nil {
		return err
	}

	// ADJUST CONTENT
	//* adjust the content for the metaphor repo
	err = AdjustMetaphorRepo(destinationMetaphorRepoURL, gitopsDir, gitProvider, gitopsTokens.MetaphorRepoName, k1Dir)
	if err != nil {
		return err
	}

	// DETOKENIZE
	//* detokenize the metaphor repo
	DetokenizeGitMetaphor(metaphorDir, metaphorTokens)
	if err != nil {
		return err
	}

	// COMMIT
	//* commit initial gitops-template content
	err = gitClient.Commit(gitopsRepo, "committing initial detokenized gitops-template repo content")
	if err != nil {
		return err
	}

	//* commit initial metaphor content
	metaphorRepo, err := git.PlainOpen(metaphorDir)
	if err != nil {
		return fmt.Errorf("error opening metaphor git repository: %s", err)
	}

	err = gitClient.Commit(metaphorRepo, "committing initial detokenized metaphor repo content")
	if err != nil {
		return err
	}

	// ADD REMOTE(S)
	//* add new remote for gitops repo
	err = gitClient.AddRemote(destinationGitopsRepoURL, gitProvider, gitopsRepo)
	if err != nil {
		return err
	}

	//* add new remote for metaphor repo
	err = gitClient.AddRemote(destinationMetaphorRepoURL, gitProvider, metaphorRepo)
	if err != nil {
		return err
	}

	return nil
}
