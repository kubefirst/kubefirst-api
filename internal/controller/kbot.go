/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"github.com/kubefirst/kubefirst-api/internal/secrets"
	pkg "github.com/kubefirst/kubefirst-api/pkg/utils"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	log "github.com/rs/zerolog/log"
)

// InitializeBot
func (clctrl *ClusterController) InitializeBot() error {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.KbotSetupCheck {
		clctrl.GitAuth.PrivateKey, clctrl.GitAuth.PublicKey, err = pkg.CreateSSHKeyPair()
		if err != nil {
			log.Error().Msgf("error generating ssh keys: %s", err)
			telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.KbotSetupFailed, err.Error())
			return err
		}

		clctrl.Cluster.GitAuth.PublicKey = clctrl.GitAuth.PublicKey
		clctrl.Cluster.GitAuth.PrivateKey = clctrl.GitAuth.PrivateKey
		clctrl.Cluster.KbotSetupCheck = true

		err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return err
		}
	}

	return nil
}
