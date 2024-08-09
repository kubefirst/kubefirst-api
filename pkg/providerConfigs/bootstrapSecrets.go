/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package providerConfigs //nolint:revive // allowing temporarily for better code organization

import (
	"context"
	"fmt"

	kube "github.com/kubefirst/kubefirst-api/internal/kubernetes"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
)

type BootstrapOptions struct {
	GitUser                  string
	DestinationGitopsRepoURL string
	GitProtocol              string
	CloudflareAPIToken       string
	CloudAuth                string
	DNSProvider              string
	CloudProvider            string
	HTTPSPassword            string
	SSHToken                 string
}

func BootstrapMgmtCluster(clientset kubernetes.Interface, opts BootstrapOptions) error {
	log.Info().Msg("creating namespaces")
	if err := Namespaces(clientset); err != nil {
		return fmt.Errorf("error creating namespaces: %w", err)
	}

	log.Info().Msg("creating service accounts")
	if err := ServiceAccounts(clientset); err != nil {
		return fmt.Errorf("error creating service accounts: %w", err)
	}

	// swap secret data based on https flag
	var secretData map[string]string

	if opts.GitProtocol == "https" {
		// http basic auth
		secretData = map[string]string{
			"type":     "git",
			"name":     opts.GitUser + "-gitops",
			"url":      opts.DestinationGitopsRepoURL,
			"username": opts.GitUser,
			"password": opts.HTTPSPassword,
		}
	} else {
		// ssh
		secretData = map[string]string{
			"type":          "git",
			"name":          opts.GitUser + "-gitops",
			"url":           opts.DestinationGitopsRepoURL,
			"sshPrivateKey": opts.SSHToken,
		}
	}

	createSecrets := []kube.Secret{
		{
			Name:        "repo-credentials-template",
			Namespace:   "argocd",
			Annotations: map[string]string{"managed-by": "argocd.argoproj.io"},
			Labels:      map[string]string{"argocd.argoproj.io/secret-type": "repository"},
			Contents:    secretData,
		},
		{
			Name:      opts.DNSProvider + "-auth",
			Namespace: "external-dns",
			Contents:  map[string]string{fmt.Sprintf("%s-auth", opts.DNSProvider): opts.CloudAuth, "cf-api-token": opts.CloudflareAPIToken},
		},
		{
			Name:      fmt.Sprintf("%s-secret", opts.CloudProvider),
			Namespace: "cert-manager",
			Contents:  map[string]string{"api-key": opts.CloudAuth},
		},
		{
			Name:      fmt.Sprintf("%s-auth", opts.DNSProvider),
			Namespace: "cert-manager",
			Contents:  map[string]string{fmt.Sprintf("%s-auth", opts.DNSProvider): opts.CloudAuth, "cf-api-token": opts.CloudflareAPIToken},
		},
	}

	if err := kube.CreateSecretsIfNotExist(context.Background(), clientset, createSecrets); err != nil {
		return fmt.Errorf("error creating secrets: %w", err)
	}

	return nil
}

func Namespaces(client kubernetes.Interface) error {
	newNamespaces := []string{
		"argocd",
		"argo",
		"atlantis",
		"chartmuseum",
		"cert-manager",
		"crossplane-system",
		"kubefirst",
		"external-dns",
		"external-secrets-operator",
		"vault",
	}

	if err := kube.CreateNamespacesIfNotExistSimple(context.Background(), client, newNamespaces); err != nil {
		return fmt.Errorf("error creating namespaces: %w", err)
	}

	return nil
}

func ServiceAccounts(client kubernetes.Interface) error {
	createServiceAccounts := []kube.ServiceAccount{
		{Name: "atlantis", Namespace: "atlantis", Automount: true},
		{Name: "external-secrets", Namespace: "external-secrets-operator", Automount: true},
	}

	if err := kube.CreateServiceAccountsIfNotExist(context.Background(), client, createServiceAccounts); err != nil {
		return fmt.Errorf("error creating service accounts: %w", err)
	}

	return nil
}

func BootstrapSecrets(client kubernetes.Interface, cl *pkgtypes.Cluster, extraSecret ...kube.Secret) error {
	var externalDNSToken string
	switch cl.DNSProvider {
	case "akamai":
		externalDNSToken = cl.AkamaiAuth.Token
	case "civo":
		externalDNSToken = cl.CivoAuth.Token
	case "vultr":
		externalDNSToken = cl.VultrAuth.Token
	case "digitalocean":
		externalDNSToken = cl.DigitaloceanAuth.Token
	case "aws":
		externalDNSToken = "implement with cluster management"
	case "google":
		externalDNSToken = "implement with cluster management"
	case "cloudflare":
		externalDNSToken = cl.CloudflareAuth.APIToken
	}

	// Create secrets
	createSecrets := []kube.Secret{
		{Name: "cloudflare-creds", Namespace: "argo", Contents: map[string]string{"origin-ca-api-key": cl.CloudflareAuth.OriginCaIssuerKey}},
		{Name: "cloudflare-creds", Namespace: "atlantis", Contents: map[string]string{"origin-ca-api-key": cl.CloudflareAuth.OriginCaIssuerKey}},
		{Name: "cloudflare-creds", Namespace: "chartmuseum", Contents: map[string]string{"origin-ca-api-key": cl.CloudflareAuth.OriginCaIssuerKey}},
		{Name: "external-dns-secrets", Namespace: "external-dns", Contents: map[string]string{"token": externalDNSToken}},
		{Name: "cloudflare-creds", Namespace: "kubefirst", Contents: map[string]string{"origin-ca-api-key": cl.CloudflareAuth.OriginCaIssuerKey}},
		{Name: "cloudflare-creds", Namespace: "vault", Contents: map[string]string{"origin-ca-api-key": cl.CloudflareAuth.OriginCaIssuerKey}},
		{Name: "kubefirst-state", Namespace: "kubefirst", Contents: map[string]string{"console-tour": "false"}},
	}

	createSecrets = append(createSecrets, extraSecret...)

	if err := kube.CreateSecretsIfNotExist(context.Background(), client, createSecrets); err != nil {
		log.Error().Msgf("error creating secrets: %s", err)
		return fmt.Errorf("error creating secrets: %w", err)
	}

	return nil
}
