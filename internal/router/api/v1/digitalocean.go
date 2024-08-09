/*
Copyright (C) 2021-2024, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/digitalocean"
	"github.com/kubefirst/kubefirst-api/internal/types"
)

func PostValidateDigitalOceanDomain(c *gin.Context) {
	domainName, exists := c.Params.Get("domain")
	if !exists {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":domain parameter not provided in request",
		})
		return
	}

	var request types.DigitalOceanDomainValidationRequest
	err := c.Bind(&request)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	digitaloceanConf := digitalocean.Configuration{
		Client:  digitalocean.NewDigitalocean(request.Token),
		Context: context.Background(),
	}

	validated := digitaloceanConf.TestDomainLiveness(domainName)
	if !validated {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: "domain validation failed",
		})
		return
	}

	c.JSON(http.StatusOK, types.DigitalOceanDomainValidationResponse{
		Validated: validated,
	})
}
