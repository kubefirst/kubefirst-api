/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package downloadManager //nolint:revive // allowing temporarily for better code organization

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kubefirst/kubefirst-api/internal/httpCommon"
	"github.com/rs/zerolog/log"
)

// DownloadFile Downloads a file from the "url" parameter, localFilename is the file destination in the local machine.
func DownloadFile(localFilename string, url string) error {
	// create local file
	out, err := os.Create(localFilename)
	if err != nil {
		return fmt.Errorf("unable to create local file %q: %s", localFilename, err)
	}
	defer out.Close()

	// get data
	resp, err := httpCommon.CustomHTTPClient(false, 0).Get(url)
	if err != nil {
		return fmt.Errorf("unable to perform GET request to %q: %s", url, err)
	}
	defer resp.Body.Close()

	// check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unable to download %q, the HTTP return status is: %s", url, resp.Status)
	}

	// writer the body to the file
	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("unable to write downloaded contents to file %q: %s", localFilename, err)
	}

	return nil
}

func ExtractFileFromTarGz(gzipStream io.Reader, tarAddress string, targetFilePath string) error {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		return fmt.Errorf("unable to create gzip reader: %w", err)
	}

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("unable to read tar contents: %w", err)
		}

		if header.Name == tarAddress {
			switch header.Typeflag {
			case tar.TypeReg:
				outFile, err := os.Create(targetFilePath)
				if err != nil {
					return fmt.Errorf("unable to create file %q: %w", targetFilePath, err)
				}
				defer outFile.Close()

				if _, err := io.Copy(outFile, tarReader); err != nil {
					return fmt.Errorf("unable to copy contents to file %q: %w", targetFilePath, err)
				}
			default:
				log.Info().Msgf("unknown type: %s in %s", string(header.Typeflag), header.Name)
			}
		}
	}

	return nil
}

func Unzip(zipFilepath string, unzipDirectory string) error {
	archive, err := zip.OpenReader(zipFilepath)
	if err != nil {
		return fmt.Errorf("unable to open zip file %q: %w", zipFilepath, err)
	}
	defer archive.Close()

	for _, f := range archive.File {
		filePath := filepath.Join(unzipDirectory, filepath.Clean(f.Name))
		log.Info().Msgf("unzipping file %s", filePath)

		if !strings.HasPrefix(filePath, filepath.Clean(unzipDirectory)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %q", filePath)
		}

		if f.FileInfo().IsDir() {
			log.Info().Msg("creating directory...")
			if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
				return fmt.Errorf("unable to create directory %q: %w", filePath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return fmt.Errorf("unable to create directory %q: %w", filepath.Dir(filePath), err)
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("unable to open file %q: %w", filePath, err)
		}
		defer dstFile.Close()

		fileInArchive, err := f.Open()
		if err != nil {
			return fmt.Errorf("unable to open file in archive %q: %w", f.Name, err)
		}
		defer fileInArchive.Close()

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			return fmt.Errorf("unable to copy file %q: %w", f.Name, err)
		}
	}

	return nil
}

func DownloadTarGz(binaryPath string, tarAddress string, targzPath string, address string) error {
	log.Info().Msgf("Downloading tar.gz from %s", address)

	if err := DownloadFile(targzPath, address); err != nil {
		return err
	}

	tarContent, err := os.Open(targzPath)
	if err != nil {
		return fmt.Errorf("unable to open file %q: %w", targzPath, err)
	}

	if err := ExtractFileFromTarGz(
		tarContent,
		tarAddress,
		binaryPath,
	); err != nil {
		return err
	}

	if err := os.Remove(targzPath); err != nil {
		return fmt.Errorf("unable to remove file %q: %w", targzPath, err)
	}

	if err := os.Chmod(binaryPath, 0o755); err != nil {
		return fmt.Errorf("unable to change file permissions %q: %w", binaryPath, err)
	}

	return nil
}

func DownloadZip(toolsDir string, address string, zipPath string) error {
	log.Info().Msgf("Downloading zip from %s", "URL")

	if err := DownloadFile(zipPath, address); err != nil {
		return err
	}

	if err := Unzip(zipPath, toolsDir); err != nil {
		return err
	}

	if err := os.RemoveAll(zipPath); err != nil {
		return fmt.Errorf("unable to remove file %q: %w", zipPath, err)
	}

	return nil
}
