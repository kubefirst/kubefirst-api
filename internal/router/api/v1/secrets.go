/*
Copyright (C) 2021-2024, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kubefirst/kubefirst-api/internal/k8s"
	"github.com/kubefirst/kubefirst-api/internal/secrets"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/kubefirst-api/internal/utils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetClusterSecret(c *gin.Context) {
	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}

	secret, param := c.Params.Get("secret")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":secret not provided",
		})
		return
	}

	kcfg := utils.GetKubernetesClient(clusterName)
	kubefirstSecrets, _ := k8s.ReadSecretV2Old(kcfg.Clientset, "kubefirst", secret)

	jsonString, err := secrets.MapToStructuredJSON(kubefirstSecrets)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, jsonString)
}

func CreateClusterSecret(c *gin.Context) {
	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}

	secretName, param := c.Params.Get("secret")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":secret not provided",
		})
		return
	}

	var secretValues map[string]interface{}
	err := c.Bind(&secretValues)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	kcfg := utils.GetKubernetesClient(clusterName)
	bytes, err := json.Marshal(secretValues)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: "error stringifying object",
		})
		return
	}

	secretValuesMap, _ := secrets.ParseJSONToMap(string(bytes))

	secretToCreate := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: "kubefirst",
		},
		Data: secretValuesMap,
	}

	err = k8s.CreateSecretV2(kcfg.Clientset, secretToCreate)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.JSONSuccessResponse{
		Message: "cluster secret created",
	})
}

func UpdateClusterSecret(c *gin.Context) {
	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}

	secret, param := c.Params.Get("secret")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":secret not provided",
		})
		return
	}

	var secretValues map[string]interface{}
	err := c.Bind(&secretValues)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	kcfg := utils.GetKubernetesClient(clusterName)

	bytes, err := json.Marshal(secretValues)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: "error stringifying object",
		})
		return
	}

	secretValuesMap, _ := secrets.ParseJSONToMap(string(bytes))
	err = k8s.UpdateSecretV2(kcfg.Clientset, "kubefirst", secret, secretValuesMap)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.JSONSuccessResponse{
		Message: "cluster secret updated",
	})
}
