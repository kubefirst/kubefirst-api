/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cloudflare

import (
	"context"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

type Configuration struct {
	Client  *cloudflare.API
	Context context.Context
}
