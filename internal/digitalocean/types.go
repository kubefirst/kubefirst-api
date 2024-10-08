/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package digitalocean

import (
	"context"

	"github.com/digitalocean/godo"
)

type Configuration struct {
	Client  *godo.Client
	Context context.Context
}

type SpacesCredentials struct {
	AccessKey       string
	SecretAccessKey string
	Endpoint        string
}
