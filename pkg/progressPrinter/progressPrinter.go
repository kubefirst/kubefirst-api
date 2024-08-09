//nolint:revive // allowing temporarily for better code organization
package progressPrinter

import internal "github.com/kubefirst/kubefirst-api/internal/progressPrinter"

var (
	IncrementTracker = internal.IncrementTracker
	AddTracker       = internal.AddTracker
	SetupProgress    = internal.SetupProgress
	TotalOfTrackers  = internal.TotalOfTrackers
	GetInstance      = internal.GetInstance
)
