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
	"strings"

	"github.com/gin-gonic/gin"
	civoruntime "github.com/kubefirst/kubefirst-api/internal/civo"
	"github.com/kubefirst/kubefirst-api/internal/constants"
	digioceanruntime "github.com/kubefirst/kubefirst-api/internal/digitalocean"
	"github.com/kubefirst/kubefirst-api/internal/env"
	environments "github.com/kubefirst/kubefirst-api/internal/environments"
	"github.com/kubefirst/kubefirst-api/internal/gitShim"
	"github.com/kubefirst/kubefirst-api/internal/k8s"
	"github.com/kubefirst/kubefirst-api/internal/secrets"
	"github.com/kubefirst/kubefirst-api/internal/services"
	"github.com/kubefirst/kubefirst-api/internal/types"
	"github.com/kubefirst/kubefirst-api/internal/utils"
	vultrruntime "github.com/kubefirst/kubefirst-api/internal/vultr"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/kubefirst-api/providers/akamai"
	"github.com/kubefirst/kubefirst-api/providers/aws"
	"github.com/kubefirst/kubefirst-api/providers/civo"
	"github.com/kubefirst/kubefirst-api/providers/digitalocean"
	"github.com/kubefirst/kubefirst-api/providers/google"
	"github.com/kubefirst/kubefirst-api/providers/k3s"
	"github.com/kubefirst/kubefirst-api/providers/vultr"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	log "github.com/rs/zerolog/log"
)

// DeleteCluster godoc
// @Summary Delete a Kubefirst cluster
// @Description Delete a Kubefirst cluster
// @Tags cluster
// @Accept json
// @Produce json
// @Param	cluster_name	path	string	true	"Cluster name"
// @Success 202 {object} types.JSONSuccessResponse
// @Failure 400 {object} types.JSONFailureResponse
// @Router /cluster/:cluster_name [delete]
// @Param Authorization header string true "API key" default(Bearer <API key>)
// DeleteCluster handles a request to delete a cluster
func DeleteCluster(c *gin.Context) {
	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}

	kcfg := utils.GetKubernetesClient(clusterName)

	// Delete cluster
	rec, err := secrets.GetCluster(kcfg.Clientset, clusterName)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	env, _ := env.GetEnv(constants.SilenceGetEnv)

	telemetryEvent := telemetry.TelemetryEvent{
		CliVersion:        env.KubefirstVersion,
		CloudProvider:     rec.CloudProvider,
		ClusterID:         rec.ClusterID,
		ClusterType:       rec.ClusterType,
		DomainName:        rec.DomainName,
		GitProvider:       rec.GitProvider,
		InstallMethod:     "",
		KubefirstClient:   "api",
		KubefirstTeam:     env.KubefirstTeam,
		KubefirstTeamInfo: env.KubefirstTeamInfo,
		MachineID:         rec.DomainName,
		ErrorMessage:      "",
		UserId:            rec.DomainName,
		MetricName:        telemetry.ClusterDeleteStarted,
	}

	if rec.LastCondition != "" {
		rec.LastCondition = ""
		err = secrets.UpdateCluster(kcfg.Clientset, rec)
		if err != nil {
			log.Warn().Msgf("error updating cluster last_condition field: %s", err)
		}
	}
	if rec.Status == constants.ClusterStatusError {
		rec.Status = constants.ClusterStatusDeleting
		err = secrets.UpdateCluster(kcfg.Clientset, rec)
		if err != nil {
			log.Warn().Msgf("error updating cluster status field: %s", err)
		}
	}

	switch rec.CloudProvider {
	case "aws":
		go func() {
			err := aws.DeleteAWSCluster(&rec, telemetryEvent)
			if err != nil {
				log.Error().Msgf(err.Error())
			}
		}()

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster delete enqueued",
		})
	case "civo":
		go func() {
			err := civo.DeleteCivoCluster(&rec, telemetryEvent)
			if err != nil {
				log.Error().Msgf(err.Error())
			}
		}()

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster delete enqueued",
		})
	case "digitalocean":
		go func() {
			err := digitalocean.DeleteDigitaloceanCluster(&rec, telemetryEvent)
			if err != nil {
				log.Error().Msgf(err.Error())
			}
		}()

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster delete enqueued",
		})
	case "vultr":
		go func() {
			err := vultr.DeleteVultrCluster(&rec, telemetryEvent)
			if err != nil {
				log.Error().Msgf(err.Error())
			}
		}()

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster delete enqueued",
		})
	case "google":
		go func() {
			err := google.DeleteGoogleCluster(&rec, telemetryEvent)
			if err != nil {
				log.Error().Msgf(err.Error())
			}
		}()

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster delete enqueued",
		})
	}
}

// GetCluster godoc
// @Summary Return a configured Kubefirst cluster
// @Description Return a configured Kubefirst cluster
// @Tags cluster
// @Accept json
// @Produce json
// @Param	cluster_name	path	string	true	"Cluster name"
// @Success 200 {object} pkgtypes.Cluster
// @Failure 400 {object} types.JSONFailureResponse
// @Router /cluster/:cluster_name [get]
// @Param Authorization header string true "API key" default(Bearer <API key>)
// GetCluster returns a specific configured cluster
func GetCluster(c *gin.Context) {
	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}

	kcfg := utils.GetKubernetesClient(clusterName)

	// Retrieve cluster info
	cluster, err := secrets.GetCluster(kcfg.Clientset, clusterName)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: "cluster not found",
		})
		return
	}

	c.JSON(http.StatusOK, cluster)
}

// GetClusters godoc
// @Summary Return all known configured Kubefirst clusters
// @Description Return all known configured Kubefirst clusters
// @Tags cluster
// @Accept json
// @Produce json
// @Success 200 {object} []pkgtypes.Cluster
// @Failure 400 {object} types.JSONFailureResponse
// @Router /cluster [get]
// @Param Authorization header string true "API key" default(Bearer <API key>)
// GetClusters returns all known configured clusters
func GetClusters(c *gin.Context) {
	kcfg := utils.GetKubernetesClient("TODO: SECRETS")

	// Retrieve all clusters info
	allClusters, err := secrets.GetClusters(kcfg.Clientset)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, allClusters)
}

// PostCreateCluster godoc
// @Summary Create a Kubefirst cluster
// @Description Create a Kubefirst cluster
// @Tags cluster
// @Accept json
// @Produce json
// @Param	cluster_name	path	string	true	"Cluster name"
// @Param	definition	body	types.ClusterDefinition	true	"Cluster create request in JSON format"
// @Success 202 {object} types.JSONSuccessResponse
// @Failure 400 {object} types.JSONFailureResponse
// @Router /cluster/:cluster_name [post]
// @Param Authorization header string true "API key" default(Bearer <API key>)
// PostCreateCluster handles a request to create a cluster
func PostCreateCluster(c *gin.Context) {
	clusterName, param := c.Params.Get("cluster_name")
	if !param || string(clusterName) == ":cluster_name" {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}
	
	// Bind to variable as application/json, handle 	error
	var clusterDefinition pkgtypes.ClusterDefinition
	err := c.Bind(&clusterDefinition)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}
	clusterDefinition.ClusterName = clusterName
	
	kcfg := utils.GetKubernetesClient(clusterName)

	// Create
	// If create is in progress, return error
	// Retrieve cluster info
	cluster, err := secrets.GetCluster(kcfg.Clientset, clusterName)
	if err != nil {
		log.Info().Msgf("cluster %s does not exist, continuing", clusterName)
	} else {
		if cluster.InProgress {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: fmt.Sprintf("%s has an active process running and another create cannot be enqeued", clusterName),
			})
			return
		}
		if cluster.LastCondition != "" {
			cluster.LastCondition = ""
			err = secrets.UpdateCluster(kcfg.Clientset, cluster)
			if err != nil {
				log.Warn().Msgf("error updating cluster last_condition field: %s", err)
			}
		}
		if cluster.Status == constants.ClusterStatusError {
			cluster.Status = constants.ClusterStatusProvisioning
			err = secrets.UpdateCluster(kcfg.Clientset, cluster)
			if err != nil {
				log.Warn().Msgf("error updating cluster status field: %s", err)
			}
		}
	}

	// Retry mechanism
	if cluster.ClusterName != "" {
		//Assign cloud and git credentials
		clusterDefinition.AkamaiAuth = cluster.AkamaiAuth
		clusterDefinition.AWSAuth = cluster.AWSAuth
		clusterDefinition.CivoAuth = cluster.CivoAuth
		clusterDefinition.VultrAuth = cluster.VultrAuth
		clusterDefinition.DigitaloceanAuth = cluster.DigitaloceanAuth
		clusterDefinition.GoogleAuth = cluster.GoogleAuth
		clusterDefinition.K3sAuth = cluster.K3sAuth
		clusterDefinition.GitAuth = cluster.GitAuth
	}

	// Determine authentication type
	useSecretForAuth := false
	k1AuthSecret := map[string]string{}

	env, _ := env.GetEnv(constants.SilenceGetEnv)

	var inCluster bool = false
	if env.InCluster == "true" {
		inCluster = true
	}

	if inCluster {
		kcfg := utils.GetKubernetesClient("")
		k1AuthSecret, err := k8s.ReadSecretV2(kcfg.Clientset, constants.KubefirstNamespace, constants.KubefirstAuthSecretName)
		if err != nil {
			log.Warn().Msgf("authentication secret does not exist, continuing: %s", err)
		} else {
			log.Info().Msg("authentication secret exists, checking contents")
			if k1AuthSecret == nil {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: "authentication secret found but contains no data, please check and try again",
				})
				return
			}
			useSecretForAuth = true
		}
	}

	switch clusterDefinition.CloudProvider {
	case "akamai":
		if useSecretForAuth {
			err := utils.ValidateAuthenticationFields(k1AuthSecret)
			if err != nil {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: fmt.Sprintf("error checking akamai auth: %s", err),
				})
				return
			}
			clusterDefinition.AkamaiAuth = pkgtypes.AkamaiAuth{
				Token: k1AuthSecret["akamai-token"],
			}
		} else {
			if clusterDefinition.AkamaiAuth.Token == "" {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: "missing authentication credentials in request, please check and try again",
				})
				return
			}
		}
		go func() {
			err = akamai.CreateAkamaiCluster(&clusterDefinition)
			if err != nil {
				log.Error().Msgf(err.Error())
			}
		}()

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster create enqueued",
		})
	case "aws":
		if useSecretForAuth {
			err := utils.ValidateAuthenticationFields(k1AuthSecret)
			if err != nil {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: fmt.Sprintf("error checking aws auth: %s", err),
				})
				return
			}
			clusterDefinition.AWSAuth = pkgtypes.AWSAuth{
				AccessKeyID:     k1AuthSecret["aws-access-key-id"],
				SecretAccessKey: k1AuthSecret["aws-secret-access-key"],
				SessionToken:    k1AuthSecret["aws-session-token"],
			}
		} else {
			if clusterDefinition.AWSAuth.AccessKeyID == "" ||
				clusterDefinition.AWSAuth.SecretAccessKey == "" ||
				clusterDefinition.AWSAuth.SessionToken == "" {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: "missing authentication credentials in request, please check and try again",
				})
				return
			}
		}
		go func() {
			err = aws.CreateAWSCluster(&clusterDefinition)
			if err != nil {
				log.Error().Msgf(err.Error())
			}
		}()

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster create enqueued",
		})
	case "civo":
		if useSecretForAuth {
			err := utils.ValidateAuthenticationFields(k1AuthSecret)
			if err != nil {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: fmt.Sprintf("error checking civo auth: %s", err),
				})
				return
			}
			clusterDefinition.CivoAuth = pkgtypes.CivoAuth{
				Token: k1AuthSecret["civo-token"],
			}
		} else {
			if clusterDefinition.CivoAuth.Token == "" {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: "missing authentication credentials in request, please check and try again",
				})
				return
			}
		}
		go func() {
			err = civo.CreateCivoCluster(&clusterDefinition)
			if err != nil {
				log.Error().Msgf(err.Error())
			}
		}()

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster create enqueued",
		})
	case "digitalocean":
		if useSecretForAuth {
			err := utils.ValidateAuthenticationFields(k1AuthSecret)
			if err != nil {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: fmt.Sprintf("error checking digitalocean auth: %s", err),
				})
				return
			}
			clusterDefinition.DigitaloceanAuth = pkgtypes.DigitaloceanAuth{
				Token:        k1AuthSecret["do-token"],
				SpacesKey:    k1AuthSecret["do-spaces-key"],
				SpacesSecret: k1AuthSecret["do-spaces-token"],
			}
		} else {
			if clusterDefinition.DigitaloceanAuth.Token == "" ||
				clusterDefinition.DigitaloceanAuth.SpacesKey == "" ||
				clusterDefinition.DigitaloceanAuth.SpacesSecret == "" {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: "missing authentication credentials in request, please check and try again",
				})
				return
			}
		}
		go func() {
			err = digitalocean.CreateDigitaloceanCluster(&clusterDefinition)
			if err != nil {
				log.Error().Msgf(err.Error())
			}
		}()

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster create enqueued",
		})
	case "vultr":
		if useSecretForAuth {
			err := utils.ValidateAuthenticationFields(k1AuthSecret)
			if err != nil {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: fmt.Sprintf("error checking vultr auth: %s", err),
				})
				return
			}
			clusterDefinition.VultrAuth = pkgtypes.VultrAuth{
				Token: k1AuthSecret["vultr-api-key"],
			}
		} else {
			if clusterDefinition.VultrAuth.Token == "" {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: "missing authentication credentials in request, please check and try again",
				})
				return
			}
		}
		go func() {
			err = vultr.CreateVultrCluster(&clusterDefinition)
			if err != nil {
				log.Error().Msgf(err.Error())
			}
		}()

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster create enqueued",
		})
	case "google":
		if useSecretForAuth {
			err := utils.ValidateAuthenticationFields(k1AuthSecret)
			if err != nil {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: fmt.Sprintf("error checking google auth: %s", err),
				})
				return
			}
			clusterDefinition.GoogleAuth = pkgtypes.GoogleAuth{
				KeyFile:   k1AuthSecret["KeyFile"],
				ProjectId: k1AuthSecret["ProjectId"],
			}
		} else {
			if clusterDefinition.GoogleAuth.KeyFile == "" {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: "missing authentication credentials in request, please check and try again",
				})
				return
			}
		}
		go func() {
			err = google.CreateGoogleCluster(&clusterDefinition)
			if err != nil {
				log.Error().Msgf(err.Error())
			}
		}()

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster create enqueued",
		})
		// }
	case "k3s":
		if useSecretForAuth {
			err := utils.ValidateAuthenticationFields(k1AuthSecret)
			if err != nil {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					Message: fmt.Sprintf("error checking k3s auth: %s", err),
				})
				return
			}
			// force empty array if not server spubilc ips provided, to avoid errror of terraform tokenisation
			defaultK3sServersPublicIps := []string{}
			if k1AuthSecret["servers-public-ips"] != "" {
				defaultK3sServersPublicIps = strings.Split(k1AuthSecret["servers-public-ips"], ",")
			}

			clusterDefinition.K3sAuth = pkgtypes.K3sAuth{
				K3sServersPrivateIps: strings.Split(k1AuthSecret["servers-private-ips"], ","),
				K3sServersPublicIps:  defaultK3sServersPublicIps,
				K3sSshUser:           k1AuthSecret["ssh-user"],
				K3sSshPrivateKey:     k1AuthSecret["ssh-privatekey"],
				K3sServersArgs:       strings.Split(k1AuthSecret["servers-args"], ","),
			}
		} else {
			if len(clusterDefinition.K3sAuth.K3sServersPrivateIps) == 0 ||
				clusterDefinition.K3sAuth.K3sSshUser == "" ||
				clusterDefinition.K3sAuth.K3sSshPrivateKey == "" {
				c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
					// Message: "missing authentication credentials in request, please check and try again",
					Message: fmt.Sprintf("missing authentication credentials in request, please check and try again: %v", clusterDefinition.K3sAuth),
				})
				return
			}
		}
		go func() {
			err = k3s.CreateK3sCluster(&clusterDefinition)
			if err != nil {
				log.Fatal().Msg(err.Error())
			}
		}()

		c.JSON(http.StatusAccepted, types.JSONSuccessResponse{
			Message: "cluster create enqueued",
		})
	}
}

// PostExportCluster godoc
// @Summary Export a Kubefirst cluster database entry
// @Description Export a Kubefirst cluster database entry
// @Tags cluster
// @Accept json
// @Produce json
// @Param	cluster_name	path	string	true	"Cluster name"
// @Success 202 {object} types.JSONSuccessResponse
// @Failure 400 {object} types.JSONFailureResponse
// @Router /cluster/:cluster_name/export [post]
// @Param Authorization header string true "API key" default(Bearer <API key>)
// PostExportCluster handles a request to export a cluster
func GetExportCluster(c *gin.Context) {
	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}

	kcfg := utils.GetKubernetesClient(clusterName)

	// get cluster object
	cluster, err := secrets.GetCluster(kcfg.Clientset, clusterName)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	// return json of cluster
	c.IndentedJSON(http.StatusOK, cluster)
}

func GetClusterKubeConfig(c *gin.Context) {

	cloudProvider, param := c.Params.Get("cloud_provider")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cloud_provider not provided",
		})
		return
	}

	var kubeConfigRequest types.KubeconfigRequest
	err := c.Bind(&kubeConfigRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	VCluster := false
	if kubeConfigRequest.VCluster {
		VCluster = true
	}

	// Handle virtual cluster kubeconfig
	if VCluster {
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "error finding home directory",
			})
			return
		}

		if kubeConfigRequest.ManagClusterName == "" {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "missing man_cluster_name",
			})
			return
		}

		kcfg := utils.GetKubernetesClient(kubeConfigRequest.ClusterName)
		internalSecret, err := k8s.ReadSecretV2(kcfg.Clientset, kubeConfigRequest.ClusterName, fmt.Sprintf("vc-%v", kubeConfigRequest.ClusterName))
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}

		config, exists := internalSecret["config"]

		if !exists {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "Unable to locate kubeconfig",
			})
			return
		}

		c.JSON(http.StatusOK, types.KubeconfigResponse{
			Config: config,
		})
		return
	}

	// handle management cluster kubeconfig
	switch cloudProvider {
	case "civo":

		if kubeConfigRequest.CivoAuth.Token == "" {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "missing civo auth token",
			})
			return
		}

		if kubeConfigRequest.CloudRegion == "" {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: "missing cloud region",
			})
			return
		}

		civoConfig := civoruntime.CivoConfiguration{
			Client:  civoruntime.NewCivo(kubeConfigRequest.CivoAuth.Token, kubeConfigRequest.CloudRegion),
			Context: context.Background(),
		}

		config, err := civoConfig.GetKubeconfig(kubeConfigRequest.ClusterName)
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, types.KubeconfigResponse{
			Config: config,
		})

	case "digitalocean":
		digitaloceanConf := digioceanruntime.DigitaloceanConfiguration{
			Client:  digioceanruntime.NewDigitalocean(kubeConfigRequest.DigitaloceanAuth.Token),
			Context: context.Background(),
		}

		config, err := digitaloceanConf.GetKubeconfig(kubeConfigRequest.ClusterName)
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, types.KubeconfigResponse{
			Config: string(config),
		})

	case "vultr":

		vultrConf := vultrruntime.VultrConfiguration{
			Client:  vultrruntime.NewVultr(kubeConfigRequest.VultrAuth.Token),
			Context: context.Background(),
		}

		config, err := vultrConf.GetKubeconfig(kubeConfigRequest.ClusterName)
		if err != nil {
			c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, types.KubeconfigResponse{
			Config: config,
		})

	default:
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("provided cloud provider: %v not implemented", cloudProvider),
		})
		return
	}
	return
}

// PostImportCluster godoc
// @Summary Import a Kubefirst cluster database entry
// @Description Import a Kubefirst cluster database entry
// @Tags cluster
// @Accept json
// @Produce json
// @Param	request_body	body	types.Cluster	true	"Cluster import request in JSON format"
// @Success 202 {object} types.JSONSuccessResponse
// @Failure 400 {object} types.JSONFailureResponse
// @Router /cluster/import [post]
// @Param Authorization header string true "API key" default(Bearer <API key>)
// PostImportCluster handles a request to import a cluster
func PostImportCluster(c *gin.Context) {
	// Bind to variable as application/json, handle error
	var cluster pkgtypes.Cluster
	err := c.Bind(&cluster)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	kcfg := utils.GetKubernetesClient(cluster.ClusterName)

	// Insert the cluster into the target database
	err = secrets.InsertCluster(kcfg.Clientset, cluster)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	// Create default service entries
	log.Info().Msg("Adding default services")
	err = services.AddDefaultServices(&cluster)
	if err != nil {
		log.Error().Msgf("error adding default service entries for cluster %s: %s", cluster.ClusterName, err)
	}

	err = gitShim.PrepareMgmtCluster(cluster)
	if err != nil {
		log.Fatal().Msgf("error cloning repository: %s", err)
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("error importing cluster %s: %s", cluster.ClusterName, err),
		})
		return
	}

	// Update cluster status in database
	cluster.InProgress = false
	err = secrets.UpdateCluster(kcfg.Clientset, cluster)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.JSONSuccessResponse{
		Message: "cluster imported",
	})
}

// PostResetClusterProgress godoc
// @Summary Remove a cluster progress marker from a cluster entry
// @Description Remove a cluster progress marker from a cluster entry
// @Tags cluster
// @Accept json
// @Produce json
// @Param	cluster_name	path	string	true	"Cluster name"
// @Success 202 {object} types.JSONSuccessResponse
// @Failure 400 {object} types.JSONFailureResponse
// @Router /cluster/:cluster_name/reset_progress [post]
// @Param Authorization header string true "API key" default(Bearer <API key>)
// PostResetClusterProgress removes a cluster progress marker from a cluster entry
func PostResetClusterProgress(c *gin.Context) {
	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}

	kcfg := utils.GetKubernetesClient(clusterName)
	// Get Cluster

	cluster, _ := secrets.GetCluster(kcfg.Clientset, clusterName)
	// Reset
	cluster.InProgress = false
	err := secrets.UpdateCluster(kcfg.Clientset, cluster)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: fmt.Sprintf("error updating cluster %s: %s", clusterName, err),
		})
		return
	}

	c.JSON(http.StatusOK, types.JSONSuccessResponse{
		Message: "cluster updated",
	})
}

// PostCreateVcluster godoc
// @Summary Create default virtual clusters
// @Description Create default virtual clusters
// @Tags cluster
// @Accept json
// @Produce json
// @Param	cluster_name	path	string	true	"Cluster name"
// @Success 202 {object} types.JSONSuccessResponse
// @Failure 400 {object} types.JSONFailureResponse
// @Router /cluster/:cluster_name/vclusters [post]
// @Param Authorization header string true "API key" default(Bearer <API key>)
// PostCreateVcluster handles a request to create default virtual cluster for the mgmt cluster
func PostCreateVcluster(c *gin.Context) {
	clusterName, param := c.Params.Get("cluster_name")
	if !param {
		c.JSON(http.StatusBadRequest, types.JSONFailureResponse{
			Message: ":cluster_name not provided",
		})
		return
	}

	kcfg := utils.GetKubernetesClient(clusterName)

	cluster, err := secrets.GetCluster(kcfg.Clientset, clusterName)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	go func() {
		err = environments.CreateDefaultClusters(cluster)
		if err != nil {
			log.Fatal().Msgf("Error creating default environments %s", err.Error())
		}
	}()

	c.JSON(http.StatusOK, types.JSONSuccessResponse{
		Message: "created default cluster environments enqueued",
	})
}
