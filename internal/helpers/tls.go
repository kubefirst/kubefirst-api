/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package helpers

import (
	"crypto/tls"
	"fmt"
)

// TestEndpointTLS determines whether or not an endpoint accepts connections over https
func TestEndpointTLS(endpoint string) error {
	_, err := tls.Dial("tcp", endpoint+":443", nil)
	if err != nil {
		return fmt.Errorf("endpoint %s doesn't support tls: %w", endpoint, err)
	}

	return nil
}
