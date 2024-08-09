/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/aws"
	"github.com/kubefirst/kubefirst-api/internal/types"
)

// GetValidateAWSDomain godoc
// @Summary Returns status of whether or not an AWS hosted zone is validated for use with Kubefirst
// @Description Returns status of whether or not an AWS hosted zone is validated for use with Kubefirst
// @Tags aws
// @Accept json
// @Produce json
// @Param	domain	path	string	true	"Domain name, no trailing dot"
// @Success 200 {object} types.AWSDomainValidateResponse
// @Failure 400 {object} types.JSONFailureResponse
// @Router /aws/domain/validate/:domain [get]
// @Param Authorization header string true "API key" default(Bearer <API key>)
// GetValidateAWSDomain returns status for an AWS domain validation
func GetValidateAWSDomain(c *gin.Context) {
	domainName, exists := c.Params.Get("domain")
	if !exists {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":domain parameter not provided in request",
		})
		return
	}

	// Run validate func
	client, err := aws.New()
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	// Requires a trailing dot for Route53
	validated, err := client.TestHostedZoneLivenessWithTxtRecords(fmt.Sprintf("%s.", domainName))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.AWSDomainValidateResponse{
		Validated: validated,
	})
}
