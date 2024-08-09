/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package utils

import (
	"fmt"
	stdLog "log"
	"os"

	pkg "github.com/kubefirst/kubefirst-api/internal"
	zeroLog "github.com/rs/zerolog"
	log "github.com/rs/zerolog/log"
)

func InitializeLogs(fileName string) error {
	homePath, err := os.UserHomeDir()
	if err != nil {
		log.Info().Msg(err.Error())
	}
	k1Dir := fmt.Sprintf("%s/.k1", homePath)

	// * create log directory
	logsFolder := fmt.Sprintf("%s/logs", k1Dir)
	_ = os.Mkdir(logsFolder, 0o700)
	if err != nil {
		return fmt.Errorf("error creating logs directory: %w", err)
	}

	// * create session log file
	logfile := fmt.Sprintf("%s/%s", logsFolder, fileName)
	logFileObj, err := pkg.OpenLogFile(logfile)
	if err != nil {
		stdLog.Panicf("unable to store log location, error is: %s - please verify the current user has write access to this directory", err)
	}

	// setup default logging
	// this Go standard log is active to keep compatibility with current code base
	stdLog.SetOutput(logFileObj)
	stdLog.SetPrefix("LOG: ")
	stdLog.SetFlags(stdLog.Ldate)

	// setup Zerolog
	log.Logger = zeroLog.New(logFileObj).With().Timestamp().Logger()

	return nil
}
