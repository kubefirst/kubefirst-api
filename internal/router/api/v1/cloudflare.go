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

	cloudflare_api "github.com/cloudflare/cloudflare-go"
	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/cloudflare"
	"github.com/kubefirst/kubefirst-api/internal/types"
)

func PostValidateCloudflareDomain(c *gin.Context) {
	domainName, exists := c.Params.Get("domain")
	if !exists {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":domain parameter not provided in request",
		})
		return
	}

	var request types.CloudflareDomainValidationRequest
	err := c.Bind(&request)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	client, err := cloudflare_api.NewWithAPIToken(request.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("Could not create cloudflare client, %v", err),
		})
		return
	}

	cloudflareConf := cloudflare.Configuration{
		Client:  client,
		Context: context.Background(),
	}

	validated := cloudflareConf.TestDomainLiveness(domainName)

	if !validated {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: "domain validation failed",
		})
		return
	}
	c.JSON(http.StatusOK, types.CloudflareDomainValidationResponse{
		Validated: validated,
	})
}
