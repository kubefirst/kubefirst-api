/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package akamai

import (
	"context"

	"github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/linode/linodego"
)

type Configuration struct {
	Client  linodego.Client
	Context context.Context
}

type BucketAndKeysConfiguration struct {
	StateStoreDetails     types.StateStoreDetails
	StateStoreCredentials types.StateStoreCredentials
}
