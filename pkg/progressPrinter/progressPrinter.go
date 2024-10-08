//nolint:revive,stylecheck // allowing temporarily for better code organization
package progressPrinter

import internal "github.com/konstructio/kubefirst-api/internal/progressPrinter"

var (
	IncrementTracker = internal.IncrementTracker
	AddTracker       = internal.AddTracker
	SetupProgress    = internal.SetupProgress
	TotalOfTrackers  = internal.TotalOfTrackers
	GetInstance      = internal.GetInstance
)
