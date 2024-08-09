/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package httpCommon //nolint:revive // allowed during code reorg

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CustomHTTPClient - creates a http client based on k1 standards
// allowInsecure defines: tls.Config{InsecureSkipVerify: allowInsecure}
func CustomHTTPClient(allowInsecure bool, conntimeout ...time.Duration) *http.Client {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()

	//nolint:gosec // allowInsecure is a user input and something we want to allow
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: allowInsecure}

	timeout := 90 * time.Minute
	if len(conntimeout) > 0 {
		timeout = conntimeout[0]
	}

	httpClient := http.Client{
		Transport: &http.Transport{
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig:       customTransport.TLSClientConfig,
		},
		Timeout: timeout,
	}
	return &httpClient
}

// ResolveAddress returns whether or not an address is resolvable
func ResolveAddress(address string) error {
	httpClient := CustomHTTPClient(false, 10*time.Second)
	resp, err := httpClient.Get(address)
	if err != nil {
		return fmt.Errorf("unable to resolve address %q: %s", address, err)
	}
	defer resp.Body.Close()

	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return fmt.Errorf("unable to read response body: %s", err)
	}

	return nil
}
