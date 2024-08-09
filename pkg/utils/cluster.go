package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	pkg "github.com/kubefirst/kubefirst-api/internal"
	"github.com/kubefirst/kubefirst-api/internal/httpCommon"
	"github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog/log"
)

// PutClusterObject exports a cluster definition as json and places it in the target object storage bucket
func PutClusterObject(cr *types.StateStoreCredentials, d *types.StateStoreDetails, obj *types.PushBucketObject) error {
	ctx := context.Background()

	// Initialize minio client
	minioClient, err := minio.New(d.Hostname, &minio.Options{
		Creds:  credentials.NewStaticV4(cr.AccessKeyID, cr.SecretAccessKey, cr.SessionToken),
		Secure: true,
	})
	if err != nil {
		return fmt.Errorf("error initializing minio client: %w", err)
	}

	// Reference for cluster object output file
	object, err := os.Open(obj.LocalFilePath)
	if err != nil {
		return fmt.Errorf("error during object local copy file lookup for %q: %w", obj.LocalFilePath, err)
	}
	defer object.Close()

	objectStat, err := object.Stat()
	if err != nil {
		return fmt.Errorf("error during object stat for %q: %w", obj.LocalFilePath, err)
	}

	// Put
	_, err = minioClient.PutObject(
		ctx,
		d.Name,
		obj.RemoteFilePath,
		object,
		objectStat.Size(),
		minio.PutObjectOptions{ContentType: obj.ContentType},
	)
	if err != nil {
		return fmt.Errorf("error during object put: %w", err)
	}
	log.Info().Msgf("uploaded cluster object %s to state store bucket %s successfully", obj.LocalFilePath, d.Name)

	if err := os.Remove(obj.LocalFilePath); err != nil {
		return fmt.Errorf("error during object local copy file removal %q: %w", obj.LocalFilePath, err)
	}

	return nil
}

// ExportCluster proxy to kubefirst api /cluster/import to restore the database
func ExportCluster(cl types.Cluster) error {
	time.Sleep(time.Second * 10)

	err := pkg.IsAppAvailable(fmt.Sprintf("%s/api/proxyHealth", pkg.KubefirstConsoleLocalURLCloud), "kubefirst api")
	if err != nil {
		log.Error().Err(err).Msg("unable to start kubefirst api")
		return fmt.Errorf("unable to start kubefirst api: %w", err)
	}

	requestObject := types.ProxyImportRequest{
		Body: cl,
		URL:  "/cluster/import",
	}

	payload, err := json.Marshal(requestObject)
	if err != nil {
		return fmt.Errorf("error marshalling request object: %w", err)
	}

	url := fmt.Sprintf("%s/api/proxy", pkg.KubefirstConsoleLocalURLCloud)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		log.Info().Msgf("error %s", err)
		return fmt.Errorf("error creating request to %q: %w", url, err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	res, err := httpCommon.CustomHTTPClient(true).Do(req)
	if err != nil {
		log.Info().Msgf("error %s", err)
		return fmt.Errorf("error sending request to %q: %w", url, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Info().Msgf("unable to export cluster %s", res.Status)
		return fmt.Errorf("unable to export cluster: status code was not 200 ok: %s", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	log.Info().Msgf("Import: %s", string(body))
	return nil
}
