/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package errors

import (
	"fmt"

	"github.com/konstructio/kubefirst-api/internal/constants"
	"github.com/konstructio/kubefirst-api/internal/secrets"
	"github.com/konstructio/kubefirst-api/internal/utils"
	pkgtypes "github.com/konstructio/kubefirst-api/pkg/types"
)

// HandleClusterError implements an error handler for standalone cluster objects
func HandleClusterError(cl *pkgtypes.Cluster, condition string) error {
	kcfg := utils.GetKubernetesClient(cl.ClusterName)

	cl.InProgress = false
	cl.Status = constants.ClusterStatusError
	cl.LastCondition = condition

	err := secrets.UpdateCluster(kcfg.Clientset, *cl)
	if err != nil {
		return fmt.Errorf("failed to update cluster %q: %w", cl.ClusterName, err)
	}

	return nil
}
