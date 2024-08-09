package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	awsinternal "github.com/kubefirst/kubefirst-api/internal/aws"
	"github.com/rs/zerolog/log"
)

func NewEKSServiceAccountClientV1() aws.Config {
	// variables are automatically available in the pod through EKS
	region := os.Getenv("AWS_REGION")
	roleArn := os.Getenv("AWS_ROLE_ARN")

	fmt.Println(fmt.Sprintf("authenticating as role arn: %s from service account", roleArn))

	awsClient, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(region),
	)
	if err != nil {
		log.Error().Msg("unable to create aws client")
	}

	return awsClient
}

type (
	Configuration       = awsinternal.Configuration
	QuotaDetailResponse = awsinternal.QuotaDetailResponse
)

var NewAwsV2 = awsinternal.NewAwsV2
