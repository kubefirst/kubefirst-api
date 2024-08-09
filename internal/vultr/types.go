/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package vultr

import (
	"context"

	"github.com/vultr/govultr/v3"
)

type Configuration struct {
	Client              *govultr.Client
	Context             context.Context
	Region              string
	ObjectStorageRegion string
}

type BucketCredentials struct {
	AccessKey       string
	SecretAccessKey string
	Endpoint        string
}
