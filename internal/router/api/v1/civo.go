/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"context"
	"net/http"

	"github.com/civo/civogo"
	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/civo"
	"github.com/kubefirst/kubefirst-api/internal/types"
)

// GetValidateCivoDomain godoc
// @Summary Returns status of whether or not a Civo hosted zone is validated for use with Kubefirst
// @Description Returns status of whether or not a Civo hosted zone is validated for use with Kubefirst
// @Tags civo
// @Accept json
// @Produce json
// @Param	domain	path	string	true	"Domain name, no trailing dot"
// @Param	settings	body	types.CivoDomainValidationRequest	true	"Domain validation request in JSON format"
// @Success 200 {object} types.CivoDomainValidationResponse
// @Failure 400 {object} types.JSONFailureResponse
// @Router /civo/domain/validate/:domain [get]
// @Param Authorization header string true "API key" default(Bearer <API key>)
// GetValidateCivoDomain returns status for a Civo domain validation
func GetValidateCivoDomain(c *gin.Context) {
	domainName, exists := c.Params.Get("domain")
	if !exists {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":domain parameter not provided in request",
		})
		return
	}

	var settings types.CivoDomainValidationRequest
	err := c.Bind(&settings)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	// Run validate func
	civoConf := civo.Configuration{
		Client:  &civogo.Client{},
		Context: context.Background(),
	}

	domainID, err := civoConf.GetDNSInfo(domainName)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	validated := civoConf.TestDomainLiveness(domainName, domainID)
	if !validated {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: "domain validation failed",
		})
		return
	}
	c.JSON(http.StatusOK, types.CivoDomainValidationResponse{
		Validated: validated,
	})
}
