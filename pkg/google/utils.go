/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package google

import (
	"fmt"
	"os"

	log "github.com/rs/zerolog/log"
)

// WriteGoogleApplicationCredentialsFile writes credentials file for use throughout installation
func WriteGoogleApplicationCredentialsFile(googleApplicationCredentials, homeDir string) error {
	file, err := os.Create(fmt.Sprintf("%s/.k1/application-default-credentials.json", homeDir))
	if err != nil {
		return err
	}

	_, err = file.WriteString(googleApplicationCredentials)
	if err != nil {
		log.Fatal().Msg("error writing google application credentials file")
		return err
	}

	// Close the file writer.
	err = file.Close()
	if err != nil {
		log.Fatal().Msg("error closing file writer")
		return err
	}
	return nil
}
