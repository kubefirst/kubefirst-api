/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package main

import (
	"fmt"

	"github.com/kubefirst/kubefirst-api/docs"
	"github.com/kubefirst/kubefirst-api/internal/env"
	api "github.com/kubefirst/kubefirst-api/internal/router"
	"github.com/kubefirst/kubefirst-api/internal/secrets"
	"github.com/kubefirst/kubefirst-api/internal/services"
	apitelemetry "github.com/kubefirst/kubefirst-api/internal/telemetry"
	"github.com/kubefirst/kubefirst-api/internal/utils"
	"github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	log "github.com/rs/zerolog/log"
)

// @title Kubefirst API
// @version 1.0
// @description Kubefirst API
// @contact.name Kubefirst
// @contact.email help@kubefirst.io
// @host localhost:port
// @BasePath /api/v1

func main() {
	env, err := env.GetEnv(false)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	log.Info().Msg("checking for cluster import secret for management cluster")
	// Import if needed
	importedCluster, err := secrets.ImportClusterIfEmpty()
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	if importedCluster.ClusterName != "" {
		log.Info().Msgf("adding default services for cluster %s", importedCluster.ClusterName)
		if err := services.AddDefaultServices(&importedCluster); err != nil {
			log.Fatal().Msg(err.Error())
		}

		if importedCluster.PostInstallCatalogApps != nil {
			go func() {
				for _, catalogApp := range importedCluster.PostInstallCatalogApps {
					log.Info().Msgf("installing catalog application %s", catalogApp.Name)

					request := &types.GitopsCatalogAppCreateRequest{
						User:       "kbot",
						SecretKeys: catalogApp.SecretKeys,
						ConfigKeys: catalogApp.ConfigKeys,
					}

					err := services.CreateService(&importedCluster, catalogApp.Name, &catalogApp, request, true)
					if err != nil {
						log.Info().Msgf("Error creating default environments %s", err.Error())
					}
				}
			}()
		}
	}

	// Programmatically set swagger info
	docs.SwaggerInfo.Title = "Kubefirst API"
	docs.SwaggerInfo.Description = "Kubefirst API"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%v", env.ServerPort)
	docs.SwaggerInfo.BasePath = "/api/v1"
	docs.SwaggerInfo.Schemes = []string{"http"}

	// Telemetry handler
	telemetryEvent := telemetry.TelemetryEvent{
		CliVersion:        env.KubefirstVersion,
		CloudProvider:     env.CloudProvider,
		ClusterID:         env.ClusterID,
		ClusterType:       env.ClusterType,
		DomainName:        env.DomainName,
		ErrorMessage:      "",
		GitProvider:       env.GitProvider,
		InstallMethod:     env.InstallMethod,
		KubefirstClient:   "api",
		KubefirstTeam:     env.KubefirstTeam,
		KubefirstTeamInfo: env.KubefirstTeamInfo,
		MachineID:         env.ClusterID,
		MetricName:        telemetry.ClusterInstallCompleted,
		ParentClusterId:   env.ParentClusterID,
		UserId:            env.ClusterID,
	}
	if env.IsClusterZero != "true" {
		// Subroutine to automatically update gitops catalog
		go utils.ScheduledGitopsCatalogUpdate()
	}
	go apitelemetry.Heartbeat(telemetryEvent)

	// API
	r := api.SetupRouter()

	err = r.Run(fmt.Sprintf(":%v", env.ServerPort))
	if err != nil {
		log.Fatal().Msgf("Error starting API: %s", err)
	}
}
