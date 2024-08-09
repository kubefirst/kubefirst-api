package reports

import (
	"bytes"
	"text/template"

	"github.com/rs/zerolog/log"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const templateHandoff = `
--------------------------------------------------------------------------------
			    !!! THIS TEXT BOX SCROLLS (use arrow keys) !!!
--------------------------------------------------------------------------------

Cluster {{ .ClusterName }} is up and running!
This information is available at $HOME/.kubefirst

Press ESC to leave this screen and return to your shell.

{{- with .MkCertClient }}
Note:
  Kubefirst generated certificates to ensure secure connections to
  your local kubernetes services. However they will not be
  trusted by your browser by default.

  To remove these warnings, you can install a new certificate
  to your local trust store by running the following command:

    {{ . }} -install

  For more details on the mkcert utility, please see:
  https://github.com/FiloSottile/mkcert
{{- end }}

--- {{ .GitProvider | caser }}
------------------------------------------------------------
  {{- if .CustomOwnerName }}
  {{ .CustomOwnerName | caser }}: {{ .GitOwner }}
  {{- else }}
  Owner: {{ .GitOwner }}
  {{- end }}
  Repos:
   {{ .DestinationGitopsRepo }}
   {{ .DestinationMetaphorRepo }}

--- Kubefirst Console
------------------------------------------------------------
  URL: http://localhost:9094/services

--- ArgoCD
------------------------------------------------------------
  URL: https://argocd.{{ .DomainName }}

--- Vault
------------------------------------------------------------
  URL: https://vault.{{ .DomainName }}

--------------------------------------------------------------------------------

Note:
  To retrieve root credentials for your kubefirst platform, including ArgoCD,
  the kbot user password, and Vault, run the following command:

	kubefirst {{ .CloudProvider }} root-credentials

  Note that this command allows you to copy these passwords directly to your
  clipboard. Provide the -h flag for additional details.

{{- with .MkCertClient }}
Note:

  The kubefirst CLI process is still running. This is a convenience feature that
  keeps port-forwarding active so you can reach your Kubernetes cluster. Before
  attempting to run any additional commands, such as "destroy", please end this
  process by pressing  ESC (escape) to release the port allocations. If you
  attempt to run any additional commands before doing so, you may get errors or
  warnings about ports already being in use.
{{- end }}
`

var tmpl = template.Must(
	template.New("handoff").Funcs(template.FuncMap{
		"caser": cases.Title(language.AmericanEnglish).String,
	}).Parse(templateHandoff),
)

type Opts struct {
	ClusterName             string
	DomainName              string
	GitOwner                string
	GitProvider             string
	DestinationGitopsRepo   string
	DestinationMetaphorRepo string
	CloudProvider           string
	MkCertClient            string
	CustomOwnerName         string
}

func renderHandoff(opts Opts, silentMode bool) {
	if silentMode {
		log.Printf("[#99] Silent mode enabled, LocalHandoffScreen skipped, please check ~/.kubefirst file for your cluster and service credentials.")
		return
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, opts); err != nil {
		log.Err(err).Msgf("failed to render handoff template: %s", err)
	}

	CommandSummary(rendered)
}
