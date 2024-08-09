/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	pkg "github.com/kubefirst/kubefirst-api/internal"
	"github.com/rs/zerolog/log"
)

// DeleteK3dCluster delete a k3d cluster
func DeleteK3dCluster(clusterName string, k1Dir string, k3dClient string) error {
	log.Info().Msgf("deleting k3d cluster %s", clusterName)
	_, _, err := pkg.ExecShellReturnStrings(k3dClient, "cluster", "delete", clusterName)
	if err != nil {
		log.Info().Msg("error deleting k3d cluster")
		return err
	}
	// todo: remove it?
	time.Sleep(20 * time.Second)

	volumeDir := fmt.Sprintf("%s/minio-storage", k1Dir)
	os.RemoveAll(volumeDir)

	return nil
}

// ResolveMinioLocal allows resolving minio over a local port forward
// useful when destroying a local lucster
func ResolveMinioLocal(path string) error {
	log.Info().Msgf("attempting to prepare terraform files pre-destroy...")
	err := filepath.Walk(path, resolveMinioLocal)
	if err != nil {
		return err
	}

	return nil
}

// resolveMinioLocal
func resolveMinioLocal(path string, fi os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if fi.IsDir() {
		return nil
	}

	read, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading file %s: %w", path, err)
	}

	newContents := string(read)
	newContents = strings.ReplaceAll(newContents, "http://minio.minio.svc.cluster.local:9000", "http://localhost:9000")

	err = os.WriteFile(path, []byte(newContents), 0)
	if err != nil {
		return fmt.Errorf("error writing file %s: %w", path, err)
	}

	return nil
}
