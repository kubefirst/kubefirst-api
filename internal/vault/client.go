/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package vault

import (
	"github.com/hashicorp/vault/api"
)

var Conf = Configuration{
	Config: NewVault(),
}

func NewVault() *api.Config {
	return api.DefaultConfig()
}
