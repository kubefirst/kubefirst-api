/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cloudflare

import (
	"context"
	"fmt"
	"net"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/konstructio/kubefirst-api/internal/dns"
	"github.com/rs/zerolog/log"
)

// GetDNSDomains lists all available DNS domains
func (c *Configuration) GetDNSDomains() ([]string, error) {
	zones, err := c.Client.ListZones(c.Context)
	if err != nil {
		return nil, fmt.Errorf("error getting cloudflare zones: %w", err)
	}

	domainList := make([]string, 0, len(zones))
	for _, domain := range zones {
		domainList = append(domainList, domain.Name)
	}

	return domainList, nil
}

// GetDNSDomains lists all available DNS domains
func (c *Configuration) GetDNSRecord() ([]string, error) {
	zones, err := c.Client.ListZones(c.Context)
	if err != nil {
		return nil, fmt.Errorf("error getting cloudflare zones: %w", err)
	}

	domainList := make([]string, 0, len(zones))
	for _, domain := range zones {
		domainList = append(domainList, domain.Name)
	}

	return domainList, nil
}

func (c *Configuration) TestDomainLiveness(domainName string) bool {
	RecordName := "kubefirst-liveness." + domainName
	RecordValue := "domain record propagated"

	// Get zone id for domain name
	zoneID, err := c.Client.ZoneIDByName(domainName)
	if err != nil {
		log.Error().Msgf("error finding cloudflare zoneid for domain %s: %s", domainName, err)
		return false
	}
	rc := cloudflare.ZoneIdentifier(zoneID)

	log.Info().Msgf("Cloudflare ZoneID %s exists and contains domain %s", zoneID, domainName)
	log.Info().Msgf("checking to see if record %s exists", domainName)

	// check for existing records

	listParams := cloudflare.ListDNSRecordsParams{
		Proxied: cloudflare.BoolPtr(false),
		Type:    "TXT",
		Name:    RecordName,
		Content: RecordValue,
	}
	existingRecords, _, err := c.Client.ListDNSRecords(c.Context, rc, listParams)
	if err != nil {
		log.Error().Msgf("error getting digitalocean dns records for domain %s: %s", domainName, err)
		return false
	}
	for _, existingRecord := range existingRecords {
		if existingRecord.Type == "TXT" && existingRecord.Name == RecordName && existingRecord.Content == RecordValue {
			log.Info().Msgf("Kubefirst DNS liveness TXT record already exists on Cloudflare")
			return true
		}
	}

	// create record if it does not exist
	createParams := cloudflare.CreateDNSRecordParams{
		TTL:     60,
		Type:    "TXT",
		Name:    RecordName,
		Content: RecordValue,
		ZoneID:  zoneID,
	}

	record, err := c.Client.CreateDNSRecord(c.Context, rc, createParams)
	if err != nil {
		log.Error().Msgf(
			"could not create kubefirst liveness TXT record on cloudflare zoneid %s for domain %s: %s",
			domainName,
			zoneID,
			err,
		)
		return false
	}
	log.Info().Msg("Kubefirst DNS liveness TXT record created on Cloudflare")

	count := 0
	// todo need to exit after n number of minutes and tell them to check ns records
	// todo this logic sucks
	for count <= 100 {
		count++

		log.Info().Msgf("%s", record.Name)
		ips, err := net.LookupTXT(fmt.Sprintf("%s.%s", record.Name, domainName))
		if err != nil {
			ips, err = dns.BackupResolver.LookupTXT(context.Background(), record.Name)
		}

		log.Info().Msgf("%s", ips)

		if err != nil {
			log.Warn().Msgf("Could not get record name %s - waiting 10 seconds and trying again: \nerror: %s", record.Name, err)
			time.Sleep(10 * time.Second)
		} else {
			for _, ip := range ips {
				// todo check ip against route53RecordValue in some capacity so we can pivot the value for testing
				log.Info().Msgf("%s. in TXT record value: %s", record.Name, ip)
				count = 101
			}
		}
		if count == 100 {
			log.Error().Msg("unable to resolve domain dns record. please check your domain registrar, ns records, and DNS host")
			return false
		}
	}
	return true
}
