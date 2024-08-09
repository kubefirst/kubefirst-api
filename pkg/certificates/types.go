/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package certificates

import "time"

// Response documents the object received from the upstream API
type Response struct {
	Query   string                   `json:"string"`
	Results []CertificateQueryResult `json:"results"`
}

// CertificateQueryResult details each object in the returned response's
// results field
type CertificateQueryResult struct {
	ID  int    `json:"crtsh_id"`
	Der string `json:"der"`
}

// CertificateDetail captures information on each individual certificate
type CertificateDetail struct {
	Issued         time.Time
	ExpirationDays float64
	DNSNames       []string
}
