/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package configs

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
)

// CheckKubefirstConfigFile validate if ~/.kubefirst file is ready to be consumed.
func CheckKubefirstConfigFile(config *Config) error {
	if _, err := os.Stat(config.KubefirstConfigFilePath); err != nil {
		e := fmt.Errorf("unable to load %q file: %w", config.KubefirstConfigFilePath, err)
		log.Error().Msg(e.Error())
		return e
	}

	log.Info().Msgf("%q file is set", config.KubefirstConfigFilePath)
	return nil
}
