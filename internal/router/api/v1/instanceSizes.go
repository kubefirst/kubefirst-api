package api

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	awsinternal "github.com/kubefirst/kubefirst-api/internal/aws"
	"github.com/kubefirst/kubefirst-api/internal/civo"
	"github.com/kubefirst/kubefirst-api/internal/digitalocean"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/kubefirst-api/internal/vultr"
	"github.com/kubefirst/kubefirst-api/pkg/aws"
	"github.com/kubefirst/kubefirst-api/pkg/google"
	"github.com/linode/linodego"
	"golang.org/x/oauth2"
)

func ListInstanceSizesForRegion(c *gin.Context) {
	dnsProvider, param := c.Params.Get("cloud_provider")

	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cloud_provider not provided",
		})
		return
	}

	var instanceSizesRequest types.InstanceSizesRequest
	err := c.Bind(&instanceSizesRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	var instanceSizesResponse types.InstanceSizesResponse

	switch dnsProvider {
	case "civo":
		if instanceSizesRequest.CivoAuth.Token == "" {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "missing civo auth token, try again",
			})
			return
		}

		civoConfig := civo.Configuration{
			Client:  civo.NewCivo(instanceSizesRequest.CivoAuth.Token, instanceSizesRequest.CloudRegion),
			Context: context.Background(),
		}

		instanceSizes, err := civoConfig.ListInstanceSizes()
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}

		instanceSizesResponse.InstanceSizes = instanceSizes
		c.JSON(http.StatusOK, instanceSizesResponse)
		return

	case "aws":
		if instanceSizesRequest.AWSAuth.AccessKeyID == "" ||
			instanceSizesRequest.AWSAuth.SecretAccessKey == "" ||
			instanceSizesRequest.AWSAuth.SessionToken == "" {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "missing authentication credentials in request, please check and try again",
			})
			return
		}

		var awsConf *awsinternal.Configuration
		if os.Getenv("IS_CLUSTER_ZERO") == "false" {
			awsConf = &awsinternal.Configuration{
				Config: aws.NewEKSServiceAccountClientV1(),
			}
		} else {
			conf, err := awsinternal.NewAwsV3(
				instanceSizesRequest.CloudRegion,
				instanceSizesRequest.AWSAuth.AccessKeyID,
				instanceSizesRequest.AWSAuth.SecretAccessKey,
				instanceSizesRequest.AWSAuth.SessionToken,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, types.JSONFailureResponse{
					Message: fmt.Sprintf("error creating aws client: %v", err),
				})
				return
			}

			awsConf = &awsinternal.Configuration{Config: conf}
		}

		instanceSizes, err := awsConf.ListInstanceSizesForRegion()
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}

		instanceSizesResponse.InstanceSizes = instanceSizes
		c.JSON(http.StatusOK, instanceSizesResponse)
		return

	case "digitalocean":
		if instanceSizesRequest.DigitaloceanAuth.Token == "" {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "missing authentication credentials in request, please check and try again",
			})
			return
		}

		digitaloceanConf := digitalocean.Configuration{
			Client:  digitalocean.NewDigitalocean(instanceSizesRequest.DigitaloceanAuth.Token),
			Context: context.Background(),
		}

		instances, err := digitaloceanConf.ListInstances()
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}

		instanceSizesResponse.InstanceSizes = instances
		c.JSON(http.StatusOK, instanceSizesResponse)
		return

	case "vultr":
		if instanceSizesRequest.VultrAuth.Token == "" {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "missing authentication credentials in request, please check and try again",
			})
			return
		}

		vultrConf := vultr.Configuration{
			Client:  vultr.NewVultr(instanceSizesRequest.VultrAuth.Token),
			Context: context.Background(),
		}

		instances, err := vultrConf.ListInstances()
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}

		instanceSizesResponse.InstanceSizes = instances
		c.JSON(http.StatusOK, instanceSizesResponse)
		return

	case "google":

		if instanceSizesRequest.CloudZone == "" {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "missing cloud_zone arg, please check and try again",
			})
			return
		}

		if instanceSizesRequest.GoogleAuth.ProjectID == "" ||
			instanceSizesRequest.GoogleAuth.KeyFile == "" {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "missing authentication credentials in request, please check and try again",
			})
			return
		}

		googleConf := google.Configuration{
			Context: context.Background(),
			Project: instanceSizesRequest.GoogleAuth.ProjectID,
			Region:  instanceSizesRequest.CloudRegion,
			KeyFile: instanceSizesRequest.GoogleAuth.KeyFile,
		}

		instances, err := googleConf.ListInstances(instanceSizesRequest.CloudZone)
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}

		instanceSizesResponse.InstanceSizes = instances

	case "akamai":
		tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: instanceSizesRequest.AkamaiAuth.Token})

		oauth2Client := &http.Client{
			Transport: &oauth2.Transport{
				Source: tokenSource,
			},
		}

		client := linodego.NewClient(oauth2Client)

		instances, err := client.ListTypes(context.Background(), &linodego.ListOptions{})
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}

		linodeInstances := []string{}

		for _, instance := range instances {
			linodeInstances = append(linodeInstances, instance.ID)
		}
		instanceSizesResponse.InstanceSizes = linodeInstances

	default:
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("unsupported dns provider: %s", dnsProvider),
		})
		return
	}

	c.JSON(http.StatusOK, instanceSizesResponse)
}
