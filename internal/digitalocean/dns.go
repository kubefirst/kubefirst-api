/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package digitalocean

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/digitalocean/godo"
	"github.com/kubefirst/kubefirst-api/internal/dns"
	"github.com/kubefirst/kubefirst-api/internal/httpCommon"
	"github.com/rs/zerolog/log"
)

func (c *Configuration) TestDomainLiveness(domainName string) bool {
	doRecordName := "kubefirst-liveness"
	doRecordValue := "domain record propagated"

	doRecordConfig := &godo.DomainRecordEditRequest{
		Name:     doRecordName,
		Type:     "TXT",
		Data:     doRecordValue,
		TTL:      600,
		Priority: *godo.PtrTo(100),
	}

	log.Info().Msgf("checking to see if record %s exists", domainName)
	log.Info().Msgf("domainName %s", domainName)

	// check for existing records
	records, err := c.GetDNSRecords(domainName)
	if err != nil {
		log.Error().Msgf("error getting digitalocean dns records for domain %s: %s", domainName, err)
		return false
	}

	for _, record := range records {
		if record.Type == "TXT" && record.Name == doRecordName {
			return true
		}
	}

	// create record if it does not exist
	_, _, err = c.Client.Domains.CreateRecord(c.Context, domainName, doRecordConfig)
	if err != nil {
		log.Warn().Msgf("%s", err)
		return false
	}
	log.Info().Msg("domain record created")

	count := 0
	// todo need to exit after n number of minutes and tell them to check ns records
	// todo this logic sucks
	for count <= 100 {
		count++

		log.Info().Msgf("%s", doRecordName)
		ips, err := net.LookupTXT(fmt.Sprintf("%s.%s", doRecordName, domainName))
		if err != nil {
			ips, err = dns.BackupResolver.LookupTXT(context.Background(), doRecordName)
		}

		log.Info().Msgf("%s", ips)

		if err != nil {
			log.Warn().Msgf("Could not get record name %s - waiting 10 seconds and trying again: \nerror: %s", doRecordName, err)
			time.Sleep(10 * time.Second)
		} else {
			for _, ip := range ips {
				// todo check ip against route53RecordValue in some capacity so we can pivot the value for testing
				log.Info().Msgf("%s. in TXT record value: %s\n", doRecordName, ip)
				count = 101
			}
		}
		if count == 100 {
			log.Error().Msg("unable to resolve domain dns record. please check your domain registrar")
			return false
		}
	}
	return true
}

// GetDNSRecords retrieves DNS records
func (c *Configuration) GetDNSRecords(domainName string) ([]godo.DomainRecord, error) {
	records, _, err := c.Client.Domains.Records(c.Context, domainName, &godo.ListOptions{})
	if err != nil {
		log.Error().Msgf("error getting digitalocean dns records for domain %s: %s", domainName, err)
		return []godo.DomainRecord{}, err
	}

	return records, nil
}

// GetDNSInfo determines whether or not a domain exists within digitalocean
func (c *Configuration) GetDNSInfo(domainName string) (string, error) {
	log.Info().Msg("GetDNSInfo (working...)")

	doDNSDomain, _, err := c.Client.Domains.Get(c.Context, domainName)
	if err != nil {
		log.Info().Msg(err.Error())
		return "", err
	}

	return doDNSDomain.Name, nil
}

// GetDomainApexContent determines whether or not a target domain features
// a host responding at zone apex
func GetDomainApexContent(domainName string) bool {
	client := httpCommon.CustomHTTPClient(false, 5*time.Second)
	exists := false
	for _, proto := range []string{"http", "https"} {
		fqdn := fmt.Sprintf("%s://%s", proto, domainName)
		_, err := client.Get(fqdn)
		if err != nil {
			log.Warn().Msgf("domain %s has no apex content", fqdn)
		} else {
			log.Info().Msgf("domain %s has apex content", fqdn)
			exists = true
		}
	}

	return exists
}

// GetDNSDomains lists all available DNS domains
func (c *Configuration) GetDNSDomains() ([]string, error) {
	domains, _, err := c.Client.Domains.List(c.Context, &godo.ListOptions{})
	if err != nil {
		return []string{}, err
	}

	domainList := make([]string, 0, len(domains))
	for _, domain := range domains {
		domainList = append(domainList, domain.Name)
	}

	return domainList, nil
}

// DeleteDNSRecords deletes provided DNS records
func (c *Configuration) DeleteDNSRecords(domainName string, dryRun bool) error {
	records, _, err := c.Client.Domains.Records(c.Context, domainName, &godo.ListOptions{})
	if err != nil {
		log.Error().Msgf("error getting digitalocean dns records for domain %s: %s", domainName, err)
		return err
	}

	for _, rec := range records {
		if rec.Type == "A" || rec.Type == "TXT" {
			msg := fmt.Sprintf("delete digitalocean dns record %s.%s [%s] %s", rec.Name, domainName, rec.Type, rec.Data)

			if dryRun {
				msg += " [dry run]"
			}

			log.Info().Msgf(msg)

			if !dryRun {
				_, err := c.Client.Domains.DeleteRecord(c.Context, domainName, rec.ID)
				if err != nil {
					return fmt.Errorf("error attempting to delete digitalocean dns record %s.%s: %s", rec.Name, domainName, err)
				}
			}
		}
	}

	return nil
}
