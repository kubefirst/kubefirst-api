package pkg

import (
	"math/rand"
	"time"

	internal "github.com/kubefirst/kubefirst-api/internal"
	helpers "github.com/kubefirst/kubefirst-api/internal/helpers"
)

func randSeq(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxy")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func Random(seq int) string {
	rand.Seed(time.Now().UnixNano())
	return randSeq(seq)
}

// internal exports
var (
	IsAppAvailable       = internal.IsAppAvailable
	GenerateClusterID    = internal.GenerateClusterID
	GetAvailableDiskSize = internal.GetAvailableDiskSize
	OpenBrowser          = internal.OpenBrowser
	ResetK1Dir           = internal.ResetK1Dir
	SetupViper           = internal.SetupViper
	OpenLogFile          = internal.OpenLogFile
)

// helper exports
var (
	DisplayLogHints       = helpers.DisplayLogHints
	TestEndpointTLS       = helpers.TestEndpointTLS
	SetClusterStatusFlags = helpers.SetClusterStatusFlags
	GetClusterStatusFlags = helpers.GetClusterStatusFlags
)
