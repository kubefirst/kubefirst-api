/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package argocdModel //nolint:revive // allowed during refactoring

type SessionSessionCreateRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
