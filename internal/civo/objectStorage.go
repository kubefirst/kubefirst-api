/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package civo

import (
	"errors"
	"fmt"
	"time"

	"github.com/civo/civogo"
	"github.com/rs/zerolog/log"
)

// CreateStorageBucket creates an object storage bucket
func (c *Configuration) CreateStorageBucket(accessKeyID string, bucketName string, region string) (*civogo.ObjectStore, error) {
	bucket, err := c.Client.NewObjectStore(&civogo.CreateObjectStoreRequest{
		Name:        bucketName,
		Region:      region,
		AccessKeyID: accessKeyID,
		MaxSizeGB:   500,
	})
	if err != nil {
		return nil, err
	}

	return bucket, nil
}

// DeleteStorageBucket deletes an object storage bucket
func (c *Configuration) DeleteStorageBucket(bucketName string) error {
	objsts, err := c.Client.ListObjectStores()
	if err != nil {
		return fmt.Errorf("error fetching object stores: %w", err)
	}

	var bucketID string

	for _, objst := range objsts.Items {
		if objst.Name == bucketName {
			bucketID = objst.ID
		}
	}

	if bucketID == "" {
		return fmt.Errorf("bucket %s not found", bucketName)
	}

	_, err = c.Client.DeleteObjectStore(bucketID)
	if err != nil {
		return fmt.Errorf("error deleting object store %s: %s", bucketName, err)
	}

	return nil
}

// GetAccessCredentials creates object store access credentials if they do not exist and returns them if they do
func (c *Configuration) GetAccessCredentials(credentialName string, region string) (*civogo.ObjectStoreCredential, error) {
	creds, err := c.checkKubefirstCredentials(credentialName)
	if err != nil && err != errNoCredsFound {
		log.Info().Msg(err.Error())
		return nil, fmt.Errorf("error fetching object store credentials: %w", err)
	}

	if err == errNoCredsFound {
		log.Info().Msgf("credential name: %s not found, creating", credentialName)
		creds, err = c.createAccessCredentials(credentialName, region)
		if err != nil {
			return nil, fmt.Errorf("error creating object store credentials: %w", err)
		}

		for i := 0; i < 12; i++ {
			creds, err = c.getAccessCredentials(creds.ID)
			if err != nil {
				return nil, fmt.Errorf("error fetching object store credentials: %w", err)
			}

			if creds.AccessKeyID != "" && creds.ID != "" && creds.Name != "" && creds.SecretAccessKeyID != "" {
				log.Info().Msgf("object storage credentials created and found after %d attempts", i+1)
				break
			}

			log.Warn().Msg("waiting for civo credentials creation")
			time.Sleep(time.Second * 10)
		}

		if creds.AccessKeyID == "" || creds.ID == "" || creds.Name == "" || creds.SecretAccessKeyID == "" {
			log.Error().Msg("Civo credentials for state bucket in object storage could not be fetched, please try to run again")
			return nil, fmt.Errorf("the Civo credentials for state bucket in object storage could not be fetched, please try to run again")
		}

		log.Info().Msgf("created object storage credential %s", credentialName)
		return creds, nil
	}

	return creds, nil
}

// DeleteAccessCredentials deletes object store credentials
func (c *Configuration) DeleteAccessCredentials(credentialName string) error {
	creds, err := c.checkKubefirstCredentials(credentialName)
	if err != nil && err != errNoCredsFound {
		log.Info().Msg(err.Error())
		return fmt.Errorf("error fetching object store credentials: %w", err)
	}

	// If no credentials are found, return
	if err == errNoCredsFound {
		return nil
	}

	_, err = c.Client.DeleteObjectStoreCredential(creds.ID)
	if err != nil {
		return fmt.Errorf("error deleting object store credentials: %w", err)
	}

	return nil
}

var errNoCredsFound = errors.New("no object store credentials found")

// checkKubefirstCredentials determines whether or not object store credentials exist
func (c *Configuration) checkKubefirstCredentials(credentialName string) (*civogo.ObjectStoreCredential, error) {
	log.Info().Msgf("looking for credential: %s", credentialName)
	remoteCredentials, err := c.Client.ListObjectStoreCredentials()
	if err != nil {
		log.Info().Msg(err.Error())
		return nil, fmt.Errorf("error fetching object store credentials: %w", err)
	}

	for i, cred := range remoteCredentials.Items {
		if cred.Name == credentialName {
			log.Info().Msgf("found credential: %s", credentialName)
			return &remoteCredentials.Items[i], nil
		}
	}

	return nil, errNoCredsFound
}

// createAccessCredentials creates access credentials for an object store
func (c *Configuration) createAccessCredentials(credentialName string, region string) (*civogo.ObjectStoreCredential, error) {
	creds, err := c.Client.NewObjectStoreCredential(&civogo.CreateObjectStoreCredentialRequest{
		Name:   credentialName,
		Region: region,
	})
	if err != nil {
		log.Info().Msgf("error creating object store credentials: %s", err.Error())
		return nil, fmt.Errorf("error creating object store credentials: %w", err)
	}
	return creds, nil
}

// getAccessCredentials retrieves an object store's access credentials
func (c *Configuration) getAccessCredentials(id string) (*civogo.ObjectStoreCredential, error) {
	creds, err := c.Client.GetObjectStoreCredential(id)
	if err != nil {
		return nil, fmt.Errorf("error fetching object store credentials: %w", err)
	}
	return creds, nil
}
