/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package certificates

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/kubefirst/kubefirst-api/pkg/reports"
)

const letsDebugHost = "https://letsdebug.net/certwatch-query"

// CheckCertificateUsage polls letsdebug to get information about used certificates
func CheckCertificateUsage(domain string) error {
	// Retrieve response from letsdebug regarding used certificates
	req, err := http.NewRequest("GET", letsDebugHost, nil)
	if err != nil {
		return err
	}
	query := fmt.Sprintf(`WITH ci AS ( SELECT min(sub.CERTIFICATE_ID) ID, min(sub.ISSUER_CA_ID) ISSUER_CA_ID, sub.CERTIFICATE DER FROM (SELECT * FROM certificate_and_identities cai WHERE plainto_tsquery('%s') @@ identities(cai.CERTIFICATE) AND cai.NAME_VALUE ILIKE ('%%' || '%s' || '%%') LIMIT 10000 ) sub GROUP BY sub.CERTIFICATE ) SELECT ci.ID crtsh_id, ci.DER der FROM ci LEFT JOIN LATERAL ( SELECT min(ctle.ENTRY_TIMESTAMP) ENTRY_TIMESTAMP FROM ct_log_entry ctle WHERE ctle.CERTIFICATE_ID = ci.ID ) le ON TRUE, ca WHERE ci.ISSUER_CA_ID = ca.ID AND x509_notBefore(ci.DER) >= NOW() - INTERVAL '169 hours' AND ci.ISSUER_CA_ID IN (16418, 183267, 183283) ORDER BY le.ENTRY_TIMESTAMP DESC;`,
		domain,
		domain,
	)
	q := req.URL.Query()
	q.Add("q", query)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var output Response

	// Decode response into struct
	err = json.NewDecoder(resp.Body).Decode(&output)
	if err != nil {
		return err
	}

	// Iterate over returned certificates
	params := make([]CertificateDetail, 0)
	for _, result := range output.Results {
		sDec, err := base64.StdEncoding.DecodeString(result.Der)
		if err != nil {
			fmt.Println(err)
		}
		cert, err := x509.ParseCertificate(sDec)
		if err != nil {
			return err
		}
		detail := CertificateDetail{
			Issued:         cert.NotBefore,
			ExpirationDays: cert.NotAfter.Sub(cert.NotBefore).Hours() / 24,
			DNSNames:       cert.DNSNames,
		}
		params = append(params, detail)
	}

	// Remove duplicates
	params = removeDuplicates(params)

	// Print
	messageHeader := fmt.Sprintf("LetsEncrypt Certificate Usage\n\nWeekly usage summary for domain %s", domain)
	message := printLetsEncryptCertData(messageHeader, params, false)
	fmt.Println(reports.StyleMessage(message))

	return nil
}

// printLetsEncryptCertData provides visual output detailing used LetsEncrypt certificates
func printLetsEncryptCertData(messageHeader string, params []CertificateDetail, showAll bool) string {
	var certificateData bytes.Buffer
	certificateData.WriteString(strings.Repeat("-", 70))
	certificateData.WriteString(fmt.Sprintf("\n%s\n\n", messageHeader))
	certificateData.WriteString(fmt.Sprintf("%v of 50 weekly certificates issued\n", len(params)))
	certificateData.WriteString(strings.Repeat("-", 70))
	certificateData.WriteString("\n\n")

	if len(params) == 0 {
		certificateData.WriteString("No certificates were retrieved.")
	}

	if !showAll {
		occurrences := make(map[string]int, 0)
		for _, cert := range params {
			existingCount, hasKey := occurrences[cert.DNSNames[0]]
			if hasKey {
				occurrences[cert.DNSNames[0]] = existingCount + 1
			} else {
				occurrences[cert.DNSNames[0]] = 1
			}
		}

		for domain, usedCertificatesCount := range occurrences {
			certificateData.WriteString(fmt.Sprintf("%s\n", domain))
			certificateData.WriteString(fmt.Sprintf("	%v of 5 of weekly certificates\n", usedCertificatesCount))
			certificateData.WriteString("")
		}
	} else {
		for _, cert := range params {
			certificateData.WriteString(fmt.Sprintf("%s:\n", cert.DNSNames[0]))
			certificateData.WriteString(fmt.Sprintf("	%s\n", cert.Issued))
			certificateData.WriteString(fmt.Sprintf("	Expires in %.0f days\n", cert.ExpirationDays))
			certificateData.WriteString("")
		}
	}

	return certificateData.String()
}

// removeDuplicates takes []CertificateDetail and removes duplicate entries
// where the DNS name and creation timestamp are identical
func removeDuplicates(sample []CertificateDetail) []CertificateDetail {
	var unique []CertificateDetail
outerLoop:
	for _, v := range sample {
		for i, u := range unique {
			if v.DNSNames[0] == u.DNSNames[0] && v.Issued == u.Issued {
				unique[i] = v
				continue outerLoop
			}
		}
		unique = append(unique, v)
	}
	return unique
}
