/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package types

// AkamaiAuth holds necessary auth credentials for interacting with civo
type AkamaiAuth struct {
	Token string `bson:"token" json:"token"`
}

// AWSAuth holds necessary auth credentials for interacting with aws
type AWSAuth struct {
	AccessKeyID     string `bson:"access_key_id" json:"access_key_id"`
	SecretAccessKey string `bson:"secret_access_key" json:"secret_access_key"`
	SessionToken    string `bson:"session_token" json:"session_token"`
}

// CivoAuth holds necessary auth credentials for interacting with civo
type CivoAuth struct {
	Token string `bson:"token" json:"token"`
}

// CloudflareAuth holds necessary auth credentials for interacting with vultr
type CloudflareAuth struct {
	Token             string `bson:"token" json:"token"` // DEPRECATED: please transition to APIToken
	APIToken          string `bson:"api_token" json:"api_token"`
	OriginCaIssuerKey string `bson:"origin_ca_issuer_key" json:"origin_ca_issuer_key"`
}

// DigitaloceanAuth holds necessary auth credentials for interacting with digitalocean
type DigitaloceanAuth struct {
	Token        string `bson:"token" json:"token"`
	SpacesKey    string `bson:"spaces_key" json:"spaces_key"`
	SpacesSecret string `bson:"spaces_secret" json:"spaces_secret"`
}

// VultrAuth holds necessary auth credentials for interacting with vultr
type VultrAuth struct {
	Token string `bson:"token" json:"token"`
}

// StateStoreCredentials
type StateStoreCredentials struct {
	AccessKeyID     string `bson:"access_key_id,omitempty" json:"access_key_id,omitempty"`
	SecretAccessKey string `bson:"secret_access_key,omitempty" json:"secret_access_key,omitempty"`
	SessionToken    string `bson:"session_token,omitempty" json:"session_token,omitempty"`
	Name            string `bson:"name,omitempty" json:"name,omitempty"`
	ID              string `bson:"id,omitempty" json:"id,omitempty"`
}

// Auth for Git Provider
type GitAuth struct {
	Token      string `bson:"git_token,omitempty" json:"git_token,omitempty"`
	User       string `bson:"git_username,omitempty" json:"git_username,omitempty"`
	Owner      string `bson:"git_owner,omitempty" json:"git_owner,omitempty"`
	PublicKey  string `bson:"public_key,omitempty" json:"public_key,omitempty"`
	PrivateKey string `bson:"private_key,omitempty" json:"private_key,omitempty"`
	PublicKeys string `bson:"public_keys,omitempty" json:"public_keys,omitempty"`
}

type VaultAuth struct {
	RootToken    string `bson:"root_token,omitempty" json:"root_token,omitempty"`
	KbotPassword string `bson:"kbot_password,omitempty" json:"kbot_password,omitempty"`
}

type GoogleAuth struct {
	KeyFile   string `bson:"key_file,omitempty" json:"key_file,omitempty"`
	ProjectID string `bson:"project_id,omitempty" json:"project_id,omitempty"`
}

type K3sAuth struct {
	K3sServersPrivateIps []string `bson:"servers_private_ips,omitempty" json:"servers_private_ips,omitempty"`
	K3sServersPublicIps  []string `bson:"servers_public_ips,omitempty" json:"servers_public_ips,omitempty"`
	K3sServersArgs       []string `bson:"servers_args,omitempty" json:"servers_args,omitempty"`
	K3sSSHUser           string   `bson:"ssh_user,omitempty" json:"ssh_user,omitempty"`
	K3sSSHPrivateKey     string   `bson:"ssh_privatekey,omitempty" json:"ssh_privatekey,omitempty"`
}
