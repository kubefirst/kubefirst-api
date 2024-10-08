/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package vault

import (
	"context"
	"fmt"
	"strings"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
)

// GetUserPassword retrieves the password for a Vault user at the users mount path
func (conf *Configuration) GetUserPassword(endpoint string, token string, username string, key string) (string, error) {
	conf.Config.Address = endpoint

	vaultClient, err := vaultapi.NewClient(conf.Config)
	if err != nil {
		return "", fmt.Errorf("error creating vault client: %w", err)
	}

	vaultClient.SetToken(token)
	if strings.Contains(endpoint, "http://") {
		vaultClient.CloneConfig().ConfigureTLS(&vaultapi.TLSConfig{
			Insecure: true,
		})
	}

	log.Info().Msg("created vault client")

	resp, err := vaultClient.KVv2("users").Get(context.Background(), username)
	if err != nil {
		return "", fmt.Errorf("error getting user %q: %w", username, err)
	}

	return resp.Data[key].(string), nil
}
