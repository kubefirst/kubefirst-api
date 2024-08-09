/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package vault

import (
	"context"
	"fmt"
	"os"
	"strings"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
)

// IterSecrets returns the contents of Vault secret data using the key/value contents of
// chosen paths in the key value store in the form of export statements that can be leveraged
// in a bash shell to set environment variables
//
// If the argument at fileName is an existing file, it will be removed
func (conf *Configuration) IterSecrets(endpoint, token, fileName string) error {
	_, err := os.Stat(fileName)
	if err != nil {
		log.Info().Msgf("file %s does not exist, continuing", fileName)
	} else {
		if err := os.Remove(fileName); err != nil {
			return fmt.Errorf("error deleting file %q: %w", fileName, err)
		}
	}

	conf.Config.Address = endpoint

	vaultClient, err := vaultapi.NewClient(conf.Config)
	if err != nil {
		return fmt.Errorf("error creating vault client: %w", err)
	}

	vaultClient.SetToken(token)
	if strings.Contains(endpoint, "http://") {
		vaultClient.CloneConfig().ConfigureTLS(&vaultapi.TLSConfig{
			Insecure: true,
		})
	}

	log.Info().Msg("created vault client")

	secretsToUse := []string{"atlantis"}
	result := make([]map[string]interface{}, 0)
	for _, s := range secretsToUse {
		resp, err := vaultClient.KVv2("secret").Get(context.Background(), s)
		if err != nil {
			return fmt.Errorf("error getting secret %q: %w", s, err)
		}

		result = append(result, resp.Data)
	}

	if _, err := os.Create(fileName); err != nil {
		return fmt.Errorf("error creating file %q: %w", fileName, err)
	}

	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("error opening file %q: %w", fileName, err)
	}
	defer f.Close()

	for _, m := range result {
		for k, v := range m {
			if k == "VAULT_ADDR" {
				_, err = f.WriteString(fmt.Sprintf("export %s=%q\n", k, endpoint))
				if err != nil {
					return fmt.Errorf("error writing to file: %w", err)
				}
			} else {
				_, err = f.WriteString(fmt.Sprintf("export %s=%q\n", k, strings.TrimSuffix(v.(string), "\n")))
				if err != nil {
					return fmt.Errorf("error writing to file: %w", err)
				}
			}
		}
	}

	return nil
}
