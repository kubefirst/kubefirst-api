/*
Copyright (C) 2021-2023, Kubefirst
This program is licensed under MIT.
See the LICENSE file for more details.
*/
package vault

import (
	"fmt"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
)

func (conf *Configuration) AutoUnseal() (*vaultapi.InitResponse, error) {
	vaultClient, err := vaultapi.NewClient(&vaultapi.Config{
		Address: VaultDefaultAddress,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating vault client: %w", err)
	}

	if err := vaultClient.CloneConfig().ConfigureTLS(&vaultapi.TLSConfig{
		Insecure: true,
	}); err != nil {
		return nil, fmt.Errorf("error configuring vault client TLS insecure flow: %w", err)
	}

	log.Info().Msg("created vault client, initializing vault with auto unseal")

	initResponse, err := vaultClient.Sys().Init(&vaultapi.InitRequest{
		RecoveryShares:    RecoveryShares,
		RecoveryThreshold: RecoveryThreshold,
	})
	if err != nil {
		return nil, fmt.Errorf("error initializing vault: %w", err)
	}

	log.Info().Msg("vault initialization complete")

	return initResponse, nil
}
