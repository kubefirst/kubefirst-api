/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"context"
	"fmt"
	"os"
	"strings"

	pkg "github.com/konstructio/kubefirst-api/internal"
	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GenerateTLSSecrets generates default certificates for k3d
func GenerateTLSSecrets(clientset kubernetes.Interface, config Config) error {
	sslPemDir := config.MkCertPemDir
	if _, err := os.Stat(sslPemDir); os.IsNotExist(err) {
		err := os.MkdirAll(sslPemDir, os.ModePerm)
		if err != nil {
			log.Info().Msgf("%s directory already exists, continuing", sslPemDir)
		}
	}

	for i, app := range pkg.GetCertificateAppList() {
		namespace := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: app.Namespace}}
		_, err := clientset.CoreV1().Namespaces().Get(context.TODO(), app.Namespace, metav1.GetOptions{})
		if err != nil {
			_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
			if err != nil {
				log.Error().Err(err).Msg("")
				return fmt.Errorf("error creating namespace for app %s: %w", app.AppName, err)
			}
			log.Info().Msgf("%d, %s", i, app.Namespace)
			log.Info().Msgf("namespace created: %s", app.Namespace)
		} else {
			log.Warn().Msgf("namespace %s already exists - skipping", app.Namespace)
		}

		// * generate certificate
		fullAppAddress := app.AppName + "." + DomainName                      // example: app-name.kubefirst.dev
		certFileName := config.MkCertPemDir + "/" + app.AppName + "-cert.pem" // example: app-name-cert.pem
		keyFileName := config.MkCertPemDir + "/" + app.AppName + "-key.pem"   // example: app-name-key.pem

		// * generate the mkcert
		log.Info().Msgf("generating certificate %s.%s on %s", app.AppName, DomainName, config.MkCertClient)
		_, _, err = pkg.ExecShellReturnStrings(
			config.MkCertClient,
			"-cert-file",
			certFileName,
			"-key-file",
			keyFileName,
			DomainName,
			fullAppAddress,
		)
		if err != nil {
			return fmt.Errorf("error executing shell command for app %s: %w", app.AppName, err)
		}

		// * read certificate files
		certPem, err := os.ReadFile(fmt.Sprintf("%s/ssl/%s/pem/%s-cert.pem", config.K1Dir, DomainName, app.AppName))
		if err != nil {
			return fmt.Errorf("error reading certificate file for app %s: %w", app.AppName, err)
		}
		keyPem, err := os.ReadFile(fmt.Sprintf("%s/ssl/%s/pem/%s-key.pem", config.K1Dir, DomainName, app.AppName))
		if err != nil {
			return fmt.Errorf("error reading key file for app %s: %w", app.AppName, err)
		}

		_, err = clientset.CoreV1().Secrets(app.Namespace).Get(context.TODO(), app.AppName, metav1.GetOptions{})
		if err == nil {
			log.Info().Msgf("kubernetes secret %s/%s already created - skipping", app.Namespace, app.AppName)
		} else if strings.Contains(err.Error(), "not found") {
			_, err = clientset.CoreV1().Secrets(app.Namespace).Create(context.TODO(), &v1.Secret{
				Type: "kubernetes.io/tls",
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-tls", app.AppName),
					Namespace: app.Namespace,
				},
				Data: map[string][]byte{
					"tls.crt": certPem,
					"tls.key": keyPem,
				},
			}, metav1.CreateOptions{})
			if err != nil {
				log.Error().Msgf("error creating kubernetes secret %s/%s: %s", app.Namespace, app.AppName, err)
				return fmt.Errorf("error creating kubernetes secret for app %s in namespace %s: %w", app.AppName, app.Namespace, err)
			}
			log.Info().Msgf("created kubernetes secret: %s/%s", app.Namespace, app.AppName)
		}
	}
	return nil
}

// GenerateSingleTLSSecret creates a single certificate for a host for k3d
func GenerateSingleTLSSecret(
	clientset kubernetes.Interface,
	config Config,
	app string,
	ns string,
) error {
	namespace := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: app}}
	_, err := clientset.CoreV1().Namespaces().Get(context.TODO(), ns, metav1.GetOptions{})
	if err != nil {
		_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
		if err != nil {
			log.Error().Err(err).Msg("")
			return fmt.Errorf("error creating namespace for app %s: %w", app, err)
		}
		log.Info().Msgf("namespace created: %s", ns)
	} else {
		log.Warn().Msgf("namespace %s already exists - skipping", ns)
	}

	// * generate certificate
	fullAppAddress := app + "." + DomainName                      // example: app-name.kubefirst.dev
	certFileName := config.MkCertPemDir + "/" + app + "-cert.pem" // example: app-name-cert.pem
	keyFileName := config.MkCertPemDir + "/" + app + "-key.pem"   // example: app-name-key.pem

	// * generate the mkcert
	log.Info().Msgf("generating certificate %s.%s on %s", app, DomainName, config.MkCertClient)
	_, _, err = pkg.ExecShellReturnStrings(
		config.MkCertClient,
		"-cert-file",
		certFileName,
		"-key-file",
		keyFileName,
		DomainName,
		fullAppAddress,
	)
	if err != nil {
		return fmt.Errorf("error executing shell command for app %s: %w", app, err)
	}

	// * read certificate files
	certPem, err := os.ReadFile(fmt.Sprintf("%s/ssl/%s/pem/%s-cert.pem", config.K1Dir, DomainName, app))
	if err != nil {
		return fmt.Errorf("error reading certificate file for app %s: %w", app, err)
	}
	keyPem, err := os.ReadFile(fmt.Sprintf("%s/ssl/%s/pem/%s-key.pem", config.K1Dir, DomainName, app))
	if err != nil {
		return fmt.Errorf("error reading key file for app %s: %w", app, err)
	}

	_, err = clientset.CoreV1().Secrets(ns).Get(context.TODO(), app, metav1.GetOptions{})
	if err == nil {
		log.Info().Msgf("kubernetes secret %s/%s already created - skipping", ns, app)
	} else if strings.Contains(err.Error(), "not found") {
		_, err = clientset.CoreV1().Secrets(ns).Create(context.TODO(), &v1.Secret{
			Type: "kubernetes.io/tls",
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-tls", app),
				Namespace: ns,
			},
			Data: map[string][]byte{
				"tls.crt": certPem,
				"tls.key": keyPem,
			},
		}, metav1.CreateOptions{})
		if err != nil {
			log.Error().Msgf("error creating kubernetes secret %s/%s: %s", ns, app, err)
			return fmt.Errorf("error creating kubernetes secret for app %s in namespace %s: %w", app, ns, err)
		}
		log.Info().Msgf("created kubernetes secret: %s/%s", ns, app)
	}

	return nil
}
