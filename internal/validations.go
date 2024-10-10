/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package internal

import (
	"fmt"

	"github.com/shirou/gopsutil/disk"
)

// GetAvailableDiskSize returns the available disk size in the user machine.
// It checks if the available disk size is enough to start an installation.
func GetAvailableDiskSize() (uint64, error) {
	usageStat, err := disk.Usage("/")
	if err != nil {
		return 0, fmt.Errorf("error getting available disk size: %w", err)
	}
	return usageStat.Free, nil
}
