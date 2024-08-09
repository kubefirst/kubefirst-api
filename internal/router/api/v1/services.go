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
	"github.com/kubefirst/kubefirst-api/internal/secrets"
	"github.com/kubefirst/kubefirst-api/internal/services"
	"github.com/kubefirst/kubefirst-api/internal/types"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"

	"github.com/kubefirst/kubefirst-api/internal/utils"
)

// GetServices godoc
// @Summary Returns a list of services for a cluster
// @Description Returns a list of services for a cluster
// @Tags services
// @Accept json
// @Produce json
// @Param	cluster_name	path	string	true	"Cluster name"
// @Success 200 {object} types.ClusterServiceList
// @Failure 400 {object} types.JSONFailureResponse
// @Router /services/:cluster_name [get]
// @Param Authorization header string true "API key" default(Bearer <API key>)
// GetServices returns a list of services for a cluster
func GetServices(c *gin.Context) {
	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}

	kcfg := utils.GetKubernetesClient(clusterName)

	// Retrieve all services info
	allServices, err := secrets.GetServices(kcfg.Clientset, clusterName)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, allServices)
}

// PostAddServiceToCluster godoc
// @Summary Add a gitops catalog application to a cluster as a service
// @Description Add a gitops catalog application to a cluster as a service
// @Tags services
// @Accept json
// @Produce json
// @Param	cluster_name	path	string	true	"Cluster name"
// @Param	service_name	path	string	true	"Service name to be added"
// @Param	definition	body	types.GitopsCatalogAppCreateRequest	true	"Service create request in JSON format"
// @Success 202 {object} types.JSONSuccessResponse
// @Failure 400 {object} types.JSONFailureResponse
// @Router /services/:cluster_name/:service_name [post]
// @Param Authorization header string true "API key" default(Bearer <API key>)
// PostAddServiceToCluster handles a request to add a service to a cluster based on a gitops catalog app
func PostAddServiceToCluster(c *gin.Context) {
	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}

	serviceName, param := c.Params.Get("service_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":service_name not provided",
		})
		return
	}

	kcfg := utils.GetKubernetesClient(clusterName)

	// Verify cluster exists
	_, err := secrets.GetCluster(kcfg.Clientset, clusterName)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: "cluster not found",
		})
		return
	}

	// Verify service is a valid option and determine if it requires secrets
	apps, err := secrets.GetGitopsCatalogApps(kcfg.Clientset)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}
	valid, hasKeys := false, false
	var appDef pkgtypes.GitopsCatalogApp
	for _, app := range apps.Apps {
		if app.Name == serviceName {
			valid = true
			appDef = app
			if app.SecretKeys != nil {
				hasKeys = true
			}
		}
	}
	if !valid {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("service %s is not valid", serviceName),
		})
		return
	}

	// Bind to variable as application/json, handle error
	var serviceDefinition pkgtypes.GitopsCatalogAppCreateRequest
	err = c.Bind(&serviceDefinition)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	// Verify any required secrets are present and not empty
	if hasKeys {
		if serviceDefinition.SecretKeys == nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: fmt.Sprintf("service %s has required secret keys that cannot be empty, check your request and try again", serviceName),
			})
			return
		}

		var providedKeys []string
		for _, key := range serviceDefinition.SecretKeys {
			providedKeys = append(providedKeys, key.Name)
		}

		for _, key := range appDef.SecretKeys {
			found := utils.FindStringInSlice(providedKeys, key.Name)
			if !found {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: fmt.Sprintf("%s is a required secret key", key.Name),
				})
				return
			}
			for _, subkey := range serviceDefinition.SecretKeys {
				if key.Name == subkey.Name {
					if subkey.Value == "" {
						c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
							Message: fmt.Sprintf("%s is a required secret key and its value cannot be empty", subkey.Name),
						})
						return
					}
				}
			}
		}
	}

	// Generate and apply
	cl, err := secrets.GetCluster(kcfg.Clientset, clusterName)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	err = services.CreateService(&cl, serviceName, &appDef, &serviceDefinition, false)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.JSONSuccessResponse{
		Message: fmt.Sprintf("service %s has been created", serviceName),
	})
}

// PostValidateService godoc
// @Summary Validate gitops catalog application
// @Description Validate a gitops catalog application so it can be deleted
// @Tags services
// @Accept json
// @Produce json
// @Param	cluster_name	path	string	true	"Cluster name"
// @Param	service_name	path	string	true	"Service name to be validated"
// @Param	definition	body	types.GitopsCatalogAppCreateRequest	true	"Service create request in JSON format"
// @Success 202 {object} types.GitopsCatalogAppValidateRequest
// @Failure 400 {object} types.JSONFailureResponse
// @Router /services/:cluster_name/:service_name/validate [post]
// @Param Authorization header string true "API key" default(Bearer <API key>)
// PostValidateService handles a request to add a service to a cluster based on a gitops catalog app
func PostValidateService(c *gin.Context) {
	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}

	serviceName, param := c.Params.Get("service_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":service_name not provided",
		})
		return
	}

	kcfg := utils.GetKubernetesClient(clusterName)

	// Verify cluster exists
	_, err := secrets.GetCluster(kcfg.Clientset, clusterName)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: "cluster not found",
		})
		return
	}

	// Verify service is a valid option and determine if it requires secrets
	apps, err := secrets.GetGitopsCatalogApps(kcfg.Clientset)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}
	valid := false
	for _, app := range apps.Apps {
		if app.Name == serviceName {
			valid = true
		}
	}

	if !valid {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("service %s is not valid", serviceName),
		})
		return
	}

	// Bind to variable as application/json, handle error
	var serviceDefinition pkgtypes.GitopsCatalogAppCreateRequest
	err = c.Bind(&serviceDefinition)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	// Generate and apply
	cl, err := secrets.GetCluster(kcfg.Clientset, clusterName)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	canDeleteService, err := services.ValidateService(&cl, serviceName, &serviceDefinition)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, pkgtypes.GitopsCatalogAppValidateRequest{
		CanDeleteService: canDeleteService,
	})
}

// DeleteServiceFromCluster godoc
// @Summary Remove a gitops catalog application from a cluster
// @Description Remove a gitops catalog application from a cluster
// @Tags services
// @Accept json
// @Produce json
// @Param	cluster_name	path	string	true	"Cluster name"
// @Param	service_name	path	string	true	"Service name to be removed"
// @Success 202 {object} types.JSONSuccessResponse
// @Failure 400 {object} types.JSONFailureResponse
// @Router /services/:cluster_name/:service_name [delete]
// @Param Authorization header string true "API key" default(Bearer <API key>)
// DeleteServiceFromCluster handles a request to remove a gitops catalog application from a cluster
func DeleteServiceFromCluster(c *gin.Context) {
	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}

	serviceName, param := c.Params.Get("service_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":service_name not provided",
		})
		return
	}

	kcfg := utils.GetKubernetesClient(clusterName)

	// Verify cluster exists
	cl, err := secrets.GetCluster(kcfg.Clientset, clusterName)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: "cluster not found",
		})
		return
	}

	// Verify service is a valid option and determine if it requires secrets
	apps, err := secrets.GetGitopsCatalogApps(kcfg.Clientset)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}
	valid := false
	for _, app := range apps.Apps {
		if app.Name == serviceName {
			valid = true
		}
	}
	if !valid {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("service %s is not valid", serviceName),
		})
		return
	}

	// Bind to variable as application/json, handle error
	var serviceDefinition pkgtypes.GitopsCatalogAppDeleteRequest
	err = c.Bind(&serviceDefinition)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	err = services.DeleteService(&cl, serviceName, serviceDefinition)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.JSONSuccessResponse{
		Message: fmt.Sprintf("service %s has been deleted", serviceName),
	})
}
