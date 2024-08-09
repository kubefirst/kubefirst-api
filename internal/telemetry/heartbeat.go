package telemetry

import (
	"time"

	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/env"
	"github.com/kubefirst/kubefirst-api/internal/secrets"
	"github.com/kubefirst/kubefirst-api/internal/utils"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
)

func Heartbeat(event telemetry.TelemetryEvent) {
	telemetry.SendEvent(event, telemetry.KubefirstHeartbeat, "")
	HeartbeatWorkloadClusters(event)

	for range time.Tick(time.Second * 300) {
		telemetry.SendEvent(event, telemetry.KubefirstHeartbeat, "")
		HeartbeatWorkloadClusters(event)
	}
}

func HeartbeatWorkloadClusters(event telemetry.TelemetryEvent) error {
	env, _ := env.GetEnv(constants.SilenceGetEnv)

	if env.IsClusterZero == "true" {
		return nil
	}

	kcfg := utils.GetKubernetesClient("")

	clusters, _ := secrets.GetClusters(kcfg.Clientset)

	for _, cluster := range clusters {
		if cluster.Status == constants.ClusterStatusProvisioned {
			for _, workloadCluster := range cluster.WorkloadClusters {
				if workloadCluster.Status == constants.ClusterStatusProvisioned {
					telemetryEvent := telemetry.TelemetryEvent{
						CliVersion:        event.CliVersion,
						CloudProvider:     workloadCluster.CloudProvider,
						ClusterID:         workloadCluster.ClusterID,
						ClusterType:       workloadCluster.ClusterType,
						DomainName:        workloadCluster.DomainName,
						GitProvider:       event.GitProvider,
						InstallMethod:     event.InstallMethod,
						KubefirstClient:   event.KubefirstClient,
						KubefirstTeam:     event.KubefirstTeam,
						KubefirstTeamInfo: event.KubefirstTeamInfo,
						MachineID:         workloadCluster.DomainName,
						ErrorMessage:      "",
						UserId:            workloadCluster.DomainName,
						MetricName:        telemetry.KubefirstHeartbeat,
					}

					telemetry.SendEvent(telemetryEvent, telemetry.KubefirstHeartbeat, "")
				}
			}
		}
	}

	return nil
}
