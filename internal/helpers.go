/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package internal

import (
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/konstructio/kubefirst-api/configs"
	"github.com/konstructio/kubefirst-api/internal/progressPrinter"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func CreateDirIfNotExist(dir string) error {
	if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
		err = os.Mkdir(dir, 0o777)
		if err != nil {
			return fmt.Errorf("unable to create directory %q: %w", dir, err)
		}
	}
	return nil
}

func RemoveSubdomainV2(domainName string) (string, error) {
	domainName = strings.TrimRight(domainName, ".")
	domainSlice := strings.Split(domainName, ".")

	if len(domainSlice) < 2 {
		return "", nil
	}

	domainName = strings.Join([]string{domainSlice[len(domainSlice)-2], domainSlice[len(domainSlice)-1]}, ".")

	return domainName, nil
}

// SetupViper handles Viper config file. If config file doesn't exist, create, in case the file is available, use it.
func SetupViper(config *configs.Config, silent bool) error {
	viperConfigFile := config.KubefirstConfigFilePath

	if _, err := os.Stat(viperConfigFile); errors.Is(err, os.ErrNotExist) {
		if !silent {
			log.Info().Msgf("Config file not found, creating a blank one: %s", viperConfigFile)
		}

		if err := os.WriteFile(viperConfigFile, []byte(""), 0o600); err != nil {
			return fmt.Errorf("unable to create blank config file, error is: %w", err)
		}
	}

	viper.SetConfigFile(viperConfigFile)
	viper.SetConfigType("yaml")
	viper.AutomaticEnv() // read in environment variables that match

	// if a config file is found, read it in.
	err := viper.ReadInConfig()
	if err != nil {
		return fmt.Errorf("unable to read config file, error is: %w", err)
	}

	if !silent {
		log.Info().Msgf("Using config file: %s", viper.ConfigFileUsed())
	}

	return nil
}

// CreateFile - Create a file with its contents
func CreateFile(fileName string, fileContent []byte) error {
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer file.Close()
	_, err = file.Write(fileContent)
	if err != nil {
		return fmt.Errorf("unable to write the file: %w", err)
	}
	return nil
}

func randSeq(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func Random(seq int) string {
	//nolint:staticcheck // will be improved in future iterations
	rand.Seed(time.Now().UnixNano())
	return randSeq(seq)
}

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

var seededRand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func GenerateClusterID() string {
	return StringWithCharset(6, charset)
}

// RemoveSubDomain receives a host and remove its subdomain, if exists.
func RemoveSubDomain(fullURL string) (string, error) {
	// add http if fullURL doesn't have it, this is for validation only, won't be used on http requests
	if !strings.HasPrefix(fullURL, "http") {
		fullURL = "https://" + fullURL
	}

	// check if received fullURL is valid before parsing it
	err := IsValidURL(fullURL)
	if err != nil {
		return "", err
	}

	// build URL
	fullPathURL, err := url.ParseRequestURI(fullURL)
	if err != nil {
		return "", fmt.Errorf("the fullURL (%s) is invalid", fullURL)
	}

	splitHost := strings.Split(fullPathURL.Host, ".")

	if len(splitHost) < 2 {
		return "", fmt.Errorf("the fullURL (%s) is invalid", fullURL)
	}

	lastURLPart := splitHost[len(splitHost)-2:]
	hostWithSpace := strings.Join(lastURLPart, " ")
	// set fullURL only without subdomain
	fullPathURL.Host = strings.ReplaceAll(hostWithSpace, " ", ".")

	// build URL without subdomain
	result := fullPathURL.Scheme + "://" + fullPathURL.Host

	// check if new URL is still valid
	err = IsValidURL(result)
	if err != nil {
		return "", err
	}

	return fullPathURL.Host, nil
}

// IsValidURL checks if a URL is valid
func IsValidURL(rawURL string) error {
	if len(rawURL) == 0 {
		return errors.New("rawURL cannot be empty string")
	}

	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil || parsedURL == nil {
		return fmt.Errorf("the URL (%s) is invalid: %w", rawURL, err)
	}

	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("the URL (%s) is invalid", rawURL)
	}

	if !parsedURL.IsAbs() {
		return fmt.Errorf("the URL (%s) is invalid: needs an absolute URL", rawURL)
	}

	return nil
}

// ValidateK1Folder receives a folder path, and expects the Kubefirst configuration folder doesn't contain "argocd-init-values.yaml" and/or "gitops/" folder.
// It follows this validation order:
//   - If folder doesn't exist, try to create it (happy path)
//   - If folder exists, and has "argocd-init-values.yaml" and/or "gitops/", abort and return error describing the issue and what should be done
func ValidateK1Folder(folderPath string) error {
	hasLeftOvers := false

	if _, err := os.Stat(folderPath); errors.Is(err, os.ErrNotExist) {
		if err = os.Mkdir(folderPath, os.ModePerm); err != nil {
			return fmt.Errorf("info: could not create directory %q - error: %w", folderPath, err)
		}
		// folder was just created, no further validation required
		return nil
	}

	_, err := os.Stat(fmt.Sprintf("%s/argocd-init-values.yaml", folderPath))
	if err == nil {
		log.Debug().Msg("found argocd-init-values.yaml file")
		hasLeftOvers = true
	}

	_, err = os.Stat(fmt.Sprintf("%s/gitops", folderPath))
	if err == nil {
		log.Debug().Msg("found git-ops path")
		hasLeftOvers = true
	}

	if hasLeftOvers {
		return fmt.Errorf("folder: %s has files that can be left overs from a previous installation, "+
			"please use kubefirst clean command to be ready for a new installation", folderPath)
	}

	return nil
}

// AwaitHostNTimes - Wait for a Host to return a 200
// - To return 200
// - To return true if host is ready, or false if not
// - Supports a number of times to test an endpoint
// - Supports the grace period after status 200 to wait before returning
func AwaitHostNTimes(url string, times int, gracePeriod time.Duration) {
	log.Printf("AwaitHostNTimes %d called with grace period of: %d seconds", times, gracePeriod)
	maxamount := times
	for i := 0; i < maxamount; i++ {
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("error: %s", err)
			time.Sleep(time.Second * 10)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Printf("%s resolved, %s second grace period required...", url, gracePeriod)
			time.Sleep(gracePeriod)
			return
		}

		log.Printf("%s not resolved, sleeping 10s", url)
		time.Sleep(time.Second * 10)
	}
}

// ReplaceFileContent receives a file path, oldContent and newContent. oldContent is the previous value that is in the
// file, newContent is the new content you want to replace.
//
// Example:
//
//	err := ReplaceFileContent(vaultMainFile, "http://127.0.0.1:9000", "http://minio.minio.svc.cluster.local:9000")
func ReplaceFileContent(filPath string, oldContent string, newContent string) error {
	file, err := os.ReadFile(filPath)
	if err != nil {
		return fmt.Errorf("unable to read file %q: %w", filPath, err)
	}

	updatedLine := strings.ReplaceAll(string(file), oldContent, newContent)

	if err = os.WriteFile(filPath, []byte(updatedLine), 0); err != nil {
		return fmt.Errorf("unable to write file %q: %w", filPath, err)
	}

	return nil
}

// UpdateTerraformS3BackendForK8sAddress during the installation process, Terraform must reach port-forwarded resources
// to be able to communicate with the services. When Kubefirst finish the installation, and Terraform needs to
// communicate with the services, it must use the internal Kubernetes addresses.
func UpdateTerraformS3BackendForK8sAddress(k1Dir string) error {
	// todo: create a function for file content replacement
	vaultMainFile := fmt.Sprintf("%s/gitops/terraform/vault/main.tf", k1Dir)
	if err := ReplaceFileContent(
		vaultMainFile,
		MinioURL,
		"http://minio.minio.svc.cluster.local:9000",
	); err != nil {
		return err
	}

	// update GitHub Terraform content
	if viper.GetString("git-provider") == "github" {
		fullPathKubefirstGitHubFile := fmt.Sprintf("%s/gitops/terraform/users/kubefirst-github.tf", k1Dir)
		if err := ReplaceFileContent(
			fullPathKubefirstGitHubFile,
			MinioURL,
			"http://minio.minio.svc.cluster.local:9000",
		); err != nil {
			return err
		}

		// change remote-backend.tf
		fullPathRemoteBackendFile := fmt.Sprintf("%s/gitops/terraform/github/remote-backend.tf", k1Dir)
		if err := ReplaceFileContent(
			fullPathRemoteBackendFile,
			MinioURL,
			"http://minio.minio.svc.cluster.local:9000",
		); err != nil {
			return err
		}
	}

	return nil
}

// UpdateTerraformS3BackendForLocalhostAddress during the destroy process, Terraform must reach port-forwarded resources
// to be able to communicate with the services.
func UpdateTerraformS3BackendForLocalhostAddress() error {
	config, err := configs.ReadConfig()
	if err != nil {
		return fmt.Errorf("unable to read config file: %w", err)
	}

	// todo: create a function for file content replacement
	vaultMainFile := fmt.Sprintf("%s/gitops/terraform/vault/main.tf", config.K1FolderPath)
	if err := ReplaceFileContent(
		vaultMainFile,
		"http://minio.minio.svc.cluster.local:9000",
		MinioURL,
	); err != nil {
		return err
	}

	gitProvider := viper.GetString("git-provider")
	// update GitHub Terraform content
	if gitProvider == "github" {
		fullPathKubefirstGitHubFile := fmt.Sprintf("%s/gitops/terraform/users/kubefirst-github.tf", config.K1FolderPath)
		if err := ReplaceFileContent(
			fullPathKubefirstGitHubFile,
			"http://minio.minio.svc.cluster.local:9000",
			MinioURL,
		); err != nil {
			return err
		}

		// change remote-backend.tf
		fullPathRemoteBackendFile := fmt.Sprintf("%s/gitops/terraform/github/remote-backend.tf", config.K1FolderPath)
		if err := ReplaceFileContent(
			fullPathRemoteBackendFile,
			"http://minio.minio.svc.cluster.local:9000",
			MinioURL,
		); err != nil {
			log.Error().Err(err).Msg("")
		}
	}

	return nil
}

// todo: deprecate cmd.informUser
func InformUser(message string, silentMode bool) {
	// if in silent mode, send message to the screen
	// silent mode will silent most of the messages, this function is not frequently called
	if silentMode {
		_, err := fmt.Fprintln(os.Stdout, message)
		if err != nil {
			log.Error().Err(err).Msg("")
		}
		return
	}
	log.Info().Msg(message)
	progressPrinter.LogMessage(fmt.Sprintf("- %s", message))
}

// OpenBrowser opens the browser with the given URL
// At this time, support is limited to darwin platforms
func OpenBrowser(url string) error {
	switch runtime.GOOS {
	case "linux":
		// if err = exec.Command("xdg-open", url).Start(); err != nil {
		//	return err
		// }
		return nil
	case "windows":
		// if err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start(); err != nil {
		//	return err
		// }
		return nil
	case "darwin":
		if err := exec.Command("open", url).Start(); err != nil {
			log.Warn().Msgf("unable to load the browser - continuing")
			return nil //nolint:nilerr // needed so the app can continue
		}
	default:
		log.Warn().Msgf("unable to load the browser, unsupported platform - continuing")
		return nil
	}

	return nil
}

// todo: this is temporary
func IsConsoleUIAvailable(url string) error {
	attempts := 10
	httpClient := http.DefaultClient
	for i := 0; i < attempts; i++ {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			log.Printf("unable to reach %q (%d/%d)", url, i+1, attempts)
			time.Sleep(5 * time.Second)
			continue
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			log.Printf("unable to reach %q (%d/%d)", url, i+1, attempts)
			time.Sleep(5 * time.Second)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Info().Msg("console UI is up and running")
			return nil
		}

		log.Info().Msg("waiting UI console to be ready")
		time.Sleep(5 * time.Second)
	}

	return nil
}

func IsAppAvailable(url string, appname string) error {
	attempts := 60
	httpClient := http.DefaultClient
	for i := 0; i < attempts; i++ {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			log.Printf("unable to reach %q (%d/%d)", url, i+1, attempts)
			time.Sleep(5 * time.Second)
			continue
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			log.Printf("unable to reach %q (%d/%d)", url, i+1, attempts)
			time.Sleep(5 * time.Second)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Info().Msgf("%s is up and running", appname)
			return nil
		}

		log.Info().Msgf("waiting %s to be ready", appname)
		time.Sleep(5 * time.Second)
	}

	return nil
}

func OpenLogFile(path string) (*os.File, error) {
	logFile, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("unable to open log file %q: %w", path, err)
	}

	return logFile, nil
}

// GetFileContent receives a file path, and return its content.
func GetFileContent(filePath string) ([]byte, error) {
	// check if file exists
	if _, err := os.Stat(filePath); err != nil && os.IsNotExist(err) {
		return nil, fmt.Errorf("file %q does not exist", filePath)
	}

	byteData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read file %q: %w", filePath, err)
	}

	return byteData, nil
}

type CertificateAppList struct {
	Namespace string
	AppName   string
}

func GetCertificateAppList() []CertificateAppList {
	certificateAppList := []CertificateAppList{
		{
			Namespace: "argo",
			AppName:   "argo",
		},
		{
			Namespace: "argocd",
			AppName:   "argocd",
		},
		{
			Namespace: "atlantis",
			AppName:   "atlantis",
		},
		{
			Namespace: "chartmuseum",
			AppName:   "chartmuseum",
		},
		{
			Namespace: "vault",
			AppName:   "vault",
		},
		{
			Namespace: "minio",
			AppName:   "minio",
		},
		{
			Namespace: "minio",
			AppName:   "minio-console",
		},
		{
			Namespace: "kubefirst",
			AppName:   "kubefirst",
		},
		{
			Namespace: "development",
			AppName:   "metaphor-development",
		},
		{
			Namespace: "staging",
			AppName:   "metaphor-staging",
		},
		{
			Namespace: "production",
			AppName:   "metaphor-production",
		},
	}

	return certificateAppList
}

// FindStringInSlice takes []string and returns true if the supplied string is in the slice.
func FindStringInSlice(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func ResetK1Dir(k1Dir string) error {
	if _, err := os.Stat(k1Dir + "/argo-workflows"); !os.IsNotExist(err) {
		// path/to/whatever exists
		err := os.RemoveAll(k1Dir + "/argo-workflows")
		if err != nil {
			return fmt.Errorf("unable to delete %q folder, error: %w", k1Dir+"/argo-workflows", err)
		}
	}

	if _, err := os.Stat(k1Dir + "/gitops"); !os.IsNotExist(err) {
		err := os.RemoveAll(k1Dir + "/gitops")
		if err != nil {
			return fmt.Errorf("unable to delete %q folder, error: %w", k1Dir+"/gitops", err)
		}
	}
	if _, err := os.Stat(k1Dir + "/metaphor"); !os.IsNotExist(err) {
		err := os.RemoveAll(k1Dir + "/metaphor")
		if err != nil {
			return fmt.Errorf("unable to delete %q folder, error: %w", k1Dir+"/metaphor", err)
		}
	}
	// todo look at logic to not re-download
	if _, err := os.Stat(k1Dir + "/tools"); !os.IsNotExist(err) {
		err = os.RemoveAll(k1Dir + "/tools")
		if err != nil {
			return fmt.Errorf("unable to delete %q folder, error: %w", k1Dir+"/tools", err)
		}
	}
	// * files
	//! this might fail with an adjustment made to validate
	if _, err := os.Stat(k1Dir + "/argocd-init-values.yaml"); !os.IsNotExist(err) {
		err = os.Remove(k1Dir + "/argocd-init-values.yaml")
		if err != nil {
			return fmt.Errorf("unable to delete %q folder, error: %w", k1Dir+"/argocd-init-values.yaml", err)
		}
	}

	return nil
}
