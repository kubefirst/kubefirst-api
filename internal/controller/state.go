/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/kubefirst/kubefirst-api/internal/civo"
	"github.com/kubefirst/kubefirst-api/internal/digitalocean"
	"github.com/kubefirst/kubefirst-api/internal/secrets"
	"github.com/kubefirst/kubefirst-api/internal/vultr"
	"github.com/kubefirst/kubefirst-api/pkg/akamai"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	"github.com/linode/linodego"
	log "github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

// StateStoreCredentials
func (clctrl *ClusterController) StateStoreCredentials() error {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return err
	}

	var stateStoreData pkgtypes.StateStoreCredentials

	telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.StateStoreCredentialsCreateStarted, "")

	if !cl.StateStoreCredsCheck {
		switch clctrl.CloudProvider {
		case "akamai":
			log.Info().Msg("object storage credentials created during bucket create")
		case "aws":
			kubefirstStateStoreBucket, err := clctrl.AwsClient.CreateBucket(clctrl.KubefirstStateStoreBucketName)
			if err != nil {
				return err
			}

			kubefirstArtifactsBucket, err := clctrl.AwsClient.CreateBucket(clctrl.KubefirstArtifactsBucketName)
			if err != nil {
				return err
			}

			stateStoreData = pkgtypes.StateStoreCredentials{
				AccessKeyID:     clctrl.AWSAuth.AccessKeyID,
				SecretAccessKey: clctrl.AWSAuth.SecretAccessKey,
				SessionToken:    clctrl.AWSAuth.SessionToken,
				Name:            clctrl.KubefirstStateStoreBucketName,
			}

			clctrl.Cluster.StateStoreDetails = pkgtypes.StateStoreDetails{
				AWSStateStoreBucket: strings.ReplaceAll(*kubefirstStateStoreBucket.Location, "/", ""),
				AWSArtifactsBucket:  strings.ReplaceAll(*kubefirstArtifactsBucket.Location, "/", ""),
				Hostname:            "s3.amazonaws.com",
				Name:                clctrl.KubefirstStateStoreBucketName,
			}
			err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
			if err != nil {
				telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.StateStoreCredentialsCreateFailed, err.Error())
				return err
			}
		case "civo":
			civoConf := civo.Configuration{
				Client:  civo.NewCivo(cl.CivoAuth.Token, cl.CloudRegion),
				Context: context.Background(),
			}

			creds, err := civoConf.GetAccessCredentials(clctrl.KubefirstStateStoreBucketName, clctrl.CloudRegion)
			if err != nil {
				telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.StateStoreCredentialsCreateFailed, err.Error())
				log.Error().Msg(err.Error())
			}

			stateStoreData = pkgtypes.StateStoreCredentials{
				AccessKeyID:     creds.AccessKeyID,
				SecretAccessKey: creds.SecretAccessKeyID,
				Name:            creds.Name,
				ID:              creds.ID,
			}
		case "digitalocean":
			digitaloceanConf := digitalocean.Configuration{
				Client:  digitalocean.NewDigitalocean(cl.DigitaloceanAuth.Token),
				Context: context.Background(),
			}

			creds := digitalocean.SpacesCredentials{
				AccessKey:       cl.DigitaloceanAuth.SpacesKey,
				SecretAccessKey: cl.DigitaloceanAuth.SpacesSecret,
				Endpoint:        fmt.Sprintf("%s.digitaloceanspaces.com", "nyc3"),
			}
			err = digitaloceanConf.CreateSpaceBucket(creds, clctrl.KubefirstStateStoreBucketName)
			if err != nil {
				msg := fmt.Sprintf("error creating spaces bucket %s: %s", clctrl.KubefirstStateStoreBucketName, err)
				log.Error().Msg(msg)
				telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.StateStoreCredentialsCreateFailed, err.Error())
				return fmt.Errorf(msg)
			}

			stateStoreData = pkgtypes.StateStoreCredentials{
				AccessKeyID:     creds.AccessKey,
				SecretAccessKey: creds.SecretAccessKey,
				Name:            clctrl.KubefirstStateStoreBucketName,
			}

			clctrl.Cluster.StateStoreDetails = pkgtypes.StateStoreDetails{
				Name:     clctrl.KubefirstStateStoreBucketName,
				Hostname: creds.Endpoint,
			}
			err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
			if err != nil {
				return err
			}

		case "google":
			// State is stored in a non s3 compliant gcs backend and thus the ADC provided will be used.

			// state store bucket created
			_, err := clctrl.GoogleClient.CreateBucket(clctrl.KubefirstStateStoreBucketName, []byte(clctrl.GoogleAuth.KeyFile))
			if err != nil {
				msg := fmt.Sprintf("error creating google bucket %s: %s", clctrl.KubefirstStateStoreBucketName, err)
				telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.StateStoreCreateFailed, msg)
				return fmt.Errorf(msg)
			}

		case "vultr":
			vultrConf := vultr.Configuration{
				Client:  vultr.NewVultr(cl.VultrAuth.Token),
				Context: context.Background(),
				Region:  cl.CloudRegion,
				// https://www.vultr.com/docs/vultr-object-storage/
				ObjectStorageRegion: "ewr",
			}

			objst, err := vultrConf.CreateObjectStorage(clctrl.KubefirstStateStoreBucketName)
			if err != nil {
				telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.StateStoreCreateFailed, err.Error())
				log.Error().Msg(err.Error())
				return err
			}
			err = vultrConf.CreateObjectStorageBucket(vultr.BucketCredentials{
				AccessKey:       objst.S3AccessKey,
				SecretAccessKey: objst.S3SecretKey,
				Endpoint:        objst.S3Hostname,
			}, clctrl.KubefirstStateStoreBucketName)
			if err != nil {
				telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.StateStoreCredentialsCreateFailed, err.Error())
				return fmt.Errorf("error creating vultr state storage bucket: %w", err)
			}

			stateStoreData = pkgtypes.StateStoreCredentials{
				AccessKeyID:     objst.S3AccessKey,
				SecretAccessKey: objst.S3SecretKey,
				Name:            objst.Label,
				ID:              objst.ID,
			}

			clctrl.Cluster.StateStoreDetails = pkgtypes.StateStoreDetails{
				Name:     objst.Label,
				ID:       objst.ID,
				Hostname: objst.S3Hostname,
			}
			err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
			if err != nil {
				return err
			}
		}

		clctrl.Cluster.StateStoreCredentials = stateStoreData
		clctrl.Cluster.StateStoreCredsCheck = true

		err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return err
		}

		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.CloudCredentialsCheckCompleted, "")
		log.Info().Msgf("%s object storage credentials created and set", clctrl.CloudProvider)
	}

	return nil
}

// StateStoreCreate
func (clctrl *ClusterController) StateStoreCreate() error {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.StateStoreCreateCheck {
		switch clctrl.CloudProvider {
		case "akamai":

			tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cl.AkamaiAuth.Token})

			oauth2Client := &http.Client{
				Transport: &oauth2.Transport{
					Source: tokenSource,
				},
			}

			linodego.NewClient(oauth2Client)

			akamaiConf := akamai.Configuration{
				Client:  linodego.NewClient(oauth2Client),
				Context: context.Background(),
			}

			telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.StateStoreCreateStarted, "")

			bucketAndCreds, err := akamaiConf.CreateObjectStorageBucketAndKeys(cl.ClusterName)
			if err != nil {
				telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.StateStoreCreateFailed, err.Error())
				log.Error().Msg(err.Error())
				return err
			}

			clctrl.Cluster.StateStoreDetails = pkgtypes.StateStoreDetails{
				Name:     bucketAndCreds.StateStoreDetails.Name,
				Hostname: bucketAndCreds.StateStoreDetails.Hostname,
			}
			clctrl.Cluster.StateStoreCreateCheck = true
			clctrl.Cluster.StateStoreCredentials = bucketAndCreds.StateStoreCredentials
			clctrl.Cluster.StateStoreCredsCheck = true

			err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
			if err != nil {
				return err
			}

			telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.StateStoreCreateCompleted, "")
			log.Info().Msgf("%s state store bucket created", clctrl.CloudProvider)
		case "civo":

			civoConf := civo.Configuration{
				Client:  civo.NewCivo(cl.CivoAuth.Token, cl.CloudRegion),
				Context: context.Background(),
			}

			telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.StateStoreCreateStarted, "")

			accessKeyID := cl.StateStoreCredentials.AccessKeyID
			log.Info().Msgf("access key id %s", accessKeyID)

			bucket, err := civoConf.CreateStorageBucket(accessKeyID, clctrl.KubefirstStateStoreBucketName, clctrl.CloudRegion)
			if err != nil {
				telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.StateStoreCreateFailed, err.Error())
				log.Error().Msg(err.Error())
				return err
			}

			stateStoreData := pkgtypes.StateStoreDetails{
				Name:     bucket.Name,
				ID:       bucket.ID,
				Hostname: bucket.BucketURL,
			}

			clctrl.Cluster.StateStoreDetails = stateStoreData
			clctrl.Cluster.StateStoreCreateCheck = true

			err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
			if err != nil {
				return err
			}

			telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.StateStoreCreateCompleted, "")
			log.Info().Msgf("%s state store bucket created", clctrl.CloudProvider)
		}
	}

	return nil
}
