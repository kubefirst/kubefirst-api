/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
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

// PostRegions godoc
// @Summary Return a list of regions for a cloud provider account
// @Description Return a list of regions for a cloud provider account
// @Tags region
// @Accept json
// @Produce json
// @Param	request	body	types.RegionListRequest	true	"Region list request in JSON format"
// @Success 200 {object} types.RegionListResponse
// @Failure 400 {object} types.JSONFailureResponse
// @Router /region/:cloud_provider [post]
// @Param Authorization header string true "API key" default(Bearer <API key>)
// PostRegions returns a list of regions for a cloud provider account
func PostRegions(c *gin.Context) {
	cloudProvider, param := c.Params.Get("cloud_provider")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cloud_provider not provided",
		})
		return
	}

	// Bind to variable as application/json, handle error
	var regionListRequest types.RegionListRequest
	err := c.Bind(&regionListRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	var regionListResponse types.RegionListResponse

	switch cloudProvider {
	case "aws":
		if regionListRequest.AWSAuth.AccessKeyID == "" ||
			regionListRequest.AWSAuth.SecretAccessKey == "" ||
			regionListRequest.AWSAuth.SessionToken == "" {
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
				regionListRequest.CloudRegion,
				regionListRequest.AWSAuth.AccessKeyID,
				regionListRequest.AWSAuth.SecretAccessKey,
				regionListRequest.AWSAuth.SessionToken,
			)
			if err != nil {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: fmt.Sprintf("error creating aws client: %s", err),
				})
				return
			}

			awsConf = &awsinternal.Configuration{Config: conf}
		}

		regions, err := awsConf.GetRegions()
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}
		regionListResponse.Regions = regions
	case "civo":
		if regionListRequest.CivoAuth.Token == "" {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "missing authentication credentials in request, please check and try again",
			})
			return
		}
		civoConf := civo.Configuration{
			Client:  civo.NewCivo(regionListRequest.CivoAuth.Token, regionListRequest.CloudRegion),
			Context: context.Background(),
		}

		regions, err := civoConf.GetRegions()
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}
		regionListResponse.Regions = regions
	case "digitalocean":
		if regionListRequest.DigitaloceanAuth.Token == "" {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "missing authentication credentials in request, please check and try again",
			})
			return
		}
		digitaloceanConf := digitalocean.Configuration{
			Client:  digitalocean.NewDigitalocean(regionListRequest.DigitaloceanAuth.Token),
			Context: context.Background(),
		}

		regions, err := digitaloceanConf.GetRegions()
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}
		regionListResponse.Regions = regions
	case "vultr":
		if regionListRequest.VultrAuth.Token == "" {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "missing authentication credentials in request, please check and try again",
			})
			return
		}
		vultrConf := vultr.Configuration{
			Client:  vultr.NewVultr(regionListRequest.VultrAuth.Token),
			Context: context.Background(),
		}

		regions, err := vultrConf.GetRegions()
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}
		regionListResponse.Regions = regions
	case "google":
		if regionListRequest.GoogleAuth.KeyFile == "" {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "missing authentication credentials in request, please check and try again",
			})
			return
		}
		googleConf := google.Configuration{
			Context: context.Background(),
			Project: regionListRequest.GoogleAuth.ProjectID,
			Region:  regionListRequest.CloudRegion,
			KeyFile: regionListRequest.GoogleAuth.KeyFile,
		}

		regions, err := googleConf.GetRegions()
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}
		regionListResponse.Regions = regions

	case "k3s":
		regionListResponse.Regions = []string{"on-premise (compatibility-mode)"}

	case "akamai":
		tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: regionListRequest.AkamaiAuth.Token})

		oauth2Client := &http.Client{
			Transport: &oauth2.Transport{
				Source: tokenSource,
			},
		}

		client := linodego.NewClient(oauth2Client)

		regions, err := client.ListRegions(context.Background(), &linodego.ListOptions{})
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}

		linodeRegions := []string{}

		for _, region := range regions {
			linodeRegions = append(linodeRegions, region.ID)
		}
		regionListResponse.Regions = linodeRegions

	default:
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("unsupported provider: %s", cloudProvider),
		})
		return
	}

	c.JSON(http.StatusOK, regionListResponse)
}
