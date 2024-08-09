package akamai

import (
	"context"
	"fmt"

	"github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/linode/linodego"
)

// CreateObjectStorageBucketAndKeys creates object store and access credentials
func (c *Configuration) CreateObjectStorageBucketAndKeys(clusterName string) (*BucketAndKeysConfiguration, error) {
	// todo get rid of hardcode default
	defaultCluster := "us-east-1"
	bucket, err := c.Client.CreateObjectStorageBucket(context.Background(), linodego.ObjectStorageBucketCreateOptions{
		Cluster: defaultCluster,
		Label:   clusterName,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create object storage bucket: %w", err)
	}

	creds, err := c.Client.CreateObjectStorageKey(context.Background(), linodego.ObjectStorageKeyCreateOptions{
		Label: clusterName,
		BucketAccess: &[]linodego.ObjectStorageKeyBucketAccess{
			{
				BucketName:  clusterName,
				Cluster:     defaultCluster,
				Permissions: "read_write",
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create object storage key: %w", err)
	}

	// todo add validation
	stateStoreData := types.StateStoreDetails{
		Name:     bucket.Label,
		ID:       bucket.Hostname,
		Hostname: bucket.Hostname,
	}

	stateStoreCredentialsData := types.StateStoreCredentials{
		AccessKeyID:     creds.AccessKey,
		SecretAccessKey: creds.SecretKey,
		Name:            bucket.Label,
	}

	return &BucketAndKeysConfiguration{stateStoreData, stateStoreCredentialsData}, nil
}
