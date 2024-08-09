/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"fmt"
	"os"

	"github.com/kubefirst/kubefirst-api/internal/downloadManager"
	"github.com/rs/zerolog/log"
)

func DownloadTools(clusterName string, gitProvider string, gitOwner string, toolsDir string, gitProtocol string) error {
	config := GetConfig(clusterName, gitProvider, gitOwner, gitProtocol)

	if _, err := os.Stat(toolsDir); os.IsNotExist(err) {
		err := os.MkdirAll(toolsDir, os.ModePerm)
		if err != nil {
			log.Info().Msgf("%s directory already exists, continuing", toolsDir)
		}
	}

	// * k3d
	k3dDownloadURL := fmt.Sprintf(
		"https://github.com/k3d-io/k3d/releases/download/%s/k3d-%s-%s",
		K3dVersion,
		LocalhostOS,
		LocalhostARCH,
	)
	err := downloadManager.DownloadFile(config.K3dClient, k3dDownloadURL)
	if err != nil {
		return fmt.Errorf("error while trying to download k3d: %w", err)
	}

	err = os.Chmod(config.K3dClient, 0o755)
	if err != nil {
		return err
	}

	// * kubectl
	kubectlDownloadURL := fmt.Sprintf(
		"https://dl.k8s.io/release/%s/bin/%s/%s/kubectl",
		KubectlVersion,
		LocalhostOS,
		LocalhostARCH,
	)

	err = downloadManager.DownloadFile(config.KubectlClient, kubectlDownloadURL)
	if err != nil {
		return fmt.Errorf("error while trying to download kubectl: %w", err)
	}

	err = os.Chmod(config.KubectlClient, 0o755)
	if err != nil {
		return err
	}

	// * mkcert
	// https: //github.com/FiloSottile/mkcert/releases/download/v1.4.4/mkcert-v1.4.4-darwin-amd64
	mkCertDownloadURL := fmt.Sprintf(
		"https://github.com/FiloSottile/mkcert/releases/download/%s/mkcert-%s-%s-%s",
		MkCertVersion,
		MkCertVersion,
		LocalhostOS,
		LocalhostARCH,
	)

	err = downloadManager.DownloadFile(config.MkCertClient, mkCertDownloadURL)
	if err != nil {
		return fmt.Errorf("error while trying to download mkcert: %w", err)
	}
	err = os.Chmod(config.MkCertClient, 0o755)
	if err != nil {
		return err
	}

	// * terraform
	terraformDownloadURL := fmt.Sprintf(
		"https://releases.hashicorp.com/terraform/%s/terraform_%s_%s_%s.zip",
		TerraformVersion,
		TerraformVersion,
		LocalhostOS,
		LocalhostARCH,
	)
	zipPath := fmt.Sprintf("%s/terraform.zip", config.ToolsDir)

	err = downloadManager.DownloadZip(config.ToolsDir, terraformDownloadURL, zipPath)
	if err != nil {
		return fmt.Errorf("error while trying to download terraform: %w", err)
	}

	return nil
}
