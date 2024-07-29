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
	"github.com/kubefirst/kubefirst-api/internal/dns"
	"github.com/rs/zerolog/log"
)

// GetDNSDomains lists all available DNS domains
func (c *CloudflareConfiguration) GetDNSDomains() ([]string, error) {
	var domainList []string

	zones, err := c.Client.ListZones(c.Context)
	if err != nil {
		return []string{}, err
	}

	for _, domain := range zones {
		domainList = append(domainList, domain.Name)
	}

	return domainList, nil
}

// GetDNSDomains lists all available DNS domains
func (c *CloudflareConfiguration) GetDNSRecord() ([]string, error) {
	var domainList []string

	zones, err := c.Client.ListZones(c.Context)
	if err != nil {
		return []string{}, err
	}

	for _, domain := range zones {
		domainList = append(domainList, domain.Name)
	}

	return domainList, nil
}

func (c *CloudflareConfiguration) TestDomainLiveness(domainName string) bool {
	RecordName := "kubefirst-liveness." + domainName
	RecordValue := "domain record propagated"

	fmt.Printf("Doamin Name is %s\n",domainName)

	// Get zone id for domain name
	zoneId, err := c.Client.ZoneIDByName(domainName)

	zoneId, err = c.Client.ZoneIDByName(domainName)
	fmt.Println(err)
	if err != nil {
		log.Error().Msgf("error finding cloudflare zoneid for domain %s: %s", domainName, err)
		return false
	}
	rc := cloudflare.ZoneIdentifier(zoneId)

	log.Info().Msgf("Cloudflare ZoneID %s exists and contains domain %s", zoneId, domainName)

	// Change this for origin certs

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
			return true
		}
	}

	// create record if it does not exist
	createParams := cloudflare.CreateDNSRecordParams{
		TTL:     60,
		Type:    "TXT",
		Name:    RecordName,
		Content: RecordValue,
		ZoneID:  zoneId,
	}

	record, err := c.Client.CreateDNSRecord(c.Context, rc, createParams)
	if err != nil {
		log.Error().Msgf(
			"could not create kubefirst liveness TXT record on cloudflare zoneid %s for domain %s: %s",
			domainName,
			zoneId,
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
				log.Info().Msgf("%s. in TXT record value: %s\n", record.Name, ip)
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
