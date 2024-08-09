/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package google

import (
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	storage "cloud.google.com/go/storage"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// CreateBucket creates a GCS bucket
func (conf *Configuration) CreateBucket(bucketName string, keyFile []byte) (*storage.BucketAttrs, error) {
	creds, err := google.CredentialsFromJSON(conf.Context, keyFile, secretmanager.DefaultAuthScopes()...)
	if err != nil {
		return nil, fmt.Errorf("could not create google storage client credentials: %w", err)
	}
	client, err := storage.NewClient(conf.Context, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("could not create google storage client: %w", err)
	}

	// Create bucket
	log.Info().Msgf("creating gcs bucket %s", bucketName)

	err = client.Bucket(bucketName).Create(conf.Context, conf.Project, &storage.BucketAttrs{})
	if err != nil {
		return nil, fmt.Errorf("error creating gcs bucket %s: %s", bucketName, err)
	}

	it := client.Buckets(conf.Context, conf.Project)
	for {
		pair, err := it.Next()
		if err == iterator.Done {
			return nil, fmt.Errorf("error fetching created bucket: %w", err)
		}
		if err != nil {
			return nil, err
		}
		if pair.Name == bucketName {
			return pair, nil
		}
	}
}

// DeleteBucket deletes a GCS bucket
func (conf *Configuration) DeleteBucket(bucketName string, keyFile []byte) error {
	creds, err := google.CredentialsFromJSON(conf.Context, keyFile, secretmanager.DefaultAuthScopes()...)
	if err != nil {
		return fmt.Errorf("could not create google storage client credentials: %w", err)
	}
	client, err := storage.NewClient(conf.Context, option.WithCredentials(creds))
	if err != nil {
		return fmt.Errorf("could not create google storage client: %w", err)
	}
	defer client.Close()

	// Create bucket
	log.Info().Msgf("deleting gcs bucket %s", bucketName)

	bucket := client.Bucket(bucketName)
	err = bucket.Delete(conf.Context)
	if err != nil {
		return fmt.Errorf("error deleting gcs bucket %s: %s", bucketName, err)
	}

	return nil
}

// ListBuckets lists all GCS buckets for a project
func (conf *Configuration) ListBuckets(keyFile []byte) ([]*storage.BucketAttrs, error) {
	creds, err := google.CredentialsFromJSON(conf.Context, keyFile, secretmanager.DefaultAuthScopes()...)
	if err != nil {
		return nil, fmt.Errorf("could not create google storage client credentials: %w", err)
	}
	client, err := storage.NewClient(conf.Context, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("could not create google storage client: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("could not create google storage client: %w", err)
	}
	defer client.Close()

	var buckets []*storage.BucketAttrs

	it := client.Buckets(conf.Context, conf.Project)
	for {
		pair, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		buckets = append(buckets, pair)
	}

	return buckets, nil
}
