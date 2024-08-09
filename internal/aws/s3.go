/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	pkg "github.com/kubefirst/kubefirst-api/internal"
	"github.com/rs/zerolog/log"
)

// CreateBucket
func (conf *Configuration) CreateBucket(bucketName string) (*s3.CreateBucketOutput, error) {
	s3Client := s3.NewFromConfig(conf.Config)
	log.Info().Msg(conf.Config.Region)

	// Determine called region and whether or not it's a valid location
	// constraint for S3
	validLocationConstraints := s3Types.BucketLocationConstraint(conf.Config.Region)
	var locationConstraint string
	for _, location := range validLocationConstraints.Values() {
		if string(location) == conf.Config.Region {
			locationConstraint = conf.Config.Region
			break
		}

		locationConstraint = "us-east-1"
	}

	// Create bucket
	log.Info().Msgf("creating s3 bucket %s with location constraint %s", bucketName, locationConstraint)
	s3CreateBucketInput := &s3.CreateBucketInput{}
	s3CreateBucketInput.Bucket = aws.String(bucketName)

	if conf.Config.Region != pkg.DefaultS3Region {
		s3CreateBucketInput.CreateBucketConfiguration = &s3Types.CreateBucketConfiguration{
			LocationConstraint: s3Types.BucketLocationConstraint(locationConstraint),
		}
	}

	bucket, err := s3Client.CreateBucket(context.Background(), s3CreateBucketInput)
	if err != nil {
		return &s3.CreateBucketOutput{}, fmt.Errorf("error creating s3 bucket %s: %s", bucketName, err)
	}

	versionConfigInput := &s3.PutBucketVersioningInput{
		Bucket: aws.String(bucketName),
		VersioningConfiguration: &s3Types.VersioningConfiguration{
			Status: s3Types.BucketVersioningStatusEnabled,
		},
	}

	_, err = s3Client.PutBucketVersioning(context.Background(), versionConfigInput)
	if err != nil {
		return &s3.CreateBucketOutput{}, fmt.Errorf("error creating s3 bucket %s: %s", bucketName, err)
	}
	return bucket, nil
}

// DeleteBucket
func (conf *Configuration) DeleteBucket(bucketName string) error {
	s3Client := s3.NewFromConfig(conf.Config)

	// Create bucket
	log.Info().Msgf("deleting s3 bucket %s", bucketName)
	s3DeleteBucketInput := &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err := s3Client.DeleteBucket(context.Background(), s3DeleteBucketInput)
	if err != nil {
		return fmt.Errorf("error deleting s3 bucket %s: %s", bucketName, err)
	}

	return nil
}

func (conf *Configuration) ListBuckets() (*s3.ListBucketsOutput, error) {
	fmt.Println("listing buckets")
	s3Client := s3.NewFromConfig(conf.Config)

	buckets, err := s3Client.ListBuckets(context.Background(), &s3.ListBucketsInput{})
	if err != nil {
		return &s3.ListBucketsOutput{}, err
	}

	return buckets, nil
}
