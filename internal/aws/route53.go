/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53Types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/kubefirst/kubefirst-api/internal/dns"
	"github.com/rs/zerolog/log"
)

// TestHostedZoneLiveness checks Route53 for the liveness test record
func (conf *Configuration) TestHostedZoneLiveness(hostedZoneName string) bool {
	route53RecordName := fmt.Sprintf("kubefirst-liveness.%s", hostedZoneName)
	route53RecordValue := "domain record propagated"

	route53Client := route53.NewFromConfig(conf.Config)

	hostedZoneID, err := conf.GetHostedZoneID(hostedZoneName)
	if err != nil {
		log.Error().Msg(err.Error())
		return false
	}

	log.Info().Msgf("checking to see if record %s exists", route53RecordName)
	log.Info().Msgf("hostedZoneId %s", hostedZoneID)
	log.Info().Msgf("route53RecordName %s", route53RecordName)

	// check for existing record
	records, err := route53Client.ListResourceRecordSets(context.Background(), &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(hostedZoneID),
	})
	if err != nil {
		log.Warn().Msgf("%s", err)
		return false
	}
	for _, r := range records.ResourceRecordSets {
		if *r.Name == fmt.Sprintf("%s.", route53RecordName) {
			log.Info().Msg("domain record found")
			return true
		}
	}

	// create record if it does not exist
	record, err := route53Client.ChangeResourceRecordSets(
		context.Background(),
		&route53.ChangeResourceRecordSetsInput{
			ChangeBatch: &route53Types.ChangeBatch{
				Changes: []route53Types.Change{
					{
						Action: "UPSERT",
						ResourceRecordSet: &route53Types.ResourceRecordSet{
							Name: aws.String(route53RecordName),
							Type: "TXT",
							ResourceRecords: []route53Types.ResourceRecord{
								{
									Value: aws.String(strconv.Quote(route53RecordValue)),
								},
							},
							TTL:           aws.Int64(10),
							Weight:        aws.Int64(100),
							SetIdentifier: aws.String("CREATE liveness check for kubefirst installation"),
						},
					},
				},
				Comment: aws.String("CREATE liveness check for kubefirst installation"),
			},
			HostedZoneId: aws.String(hostedZoneID),
		})
	if err != nil {
		log.Warn().Msgf("%s", err)
		return false
	}
	log.Info().Msgf("record creation status is %s", record.ChangeInfo.Status)

	count := 0
	// todo need to exit after n number of minutes and tell them to check ns records
	// todo this logic sucks
	for count <= 100 {
		count++

		log.Info().Msgf("%s", route53RecordName)
		ips, err := net.LookupTXT(route53RecordName)
		if err != nil {
			ips, err = dns.BackupResolver.LookupTXT(context.Background(), route53RecordName)
		}

		log.Info().Msgf("%s", ips)

		if err != nil {
			log.Warn().Msgf("could not get record name %s - waiting 10 seconds and trying again: \nerror: %s", route53RecordName, err)
			time.Sleep(10 * time.Second)
		} else {
			for _, ip := range ips {
				// todo check ip against route53RecordValue in some capacity so we can pivot the value for testing
				log.Info().Msgf("%s. in TXT record value: %s\n", route53RecordName, ip)
				count = 101
			}
		}
		if count == 100 {
			log.Error().Msg("unable to resolve hosted zone dns record. please check your domain registrar")
			return false
		}
	}
	return true
}

// GetHostedZoneID returns the ID of a hosted zone if valid
func (conf *Configuration) GetHostedZoneID(hostedZoneName string) (string, error) {
	route53Client := route53.NewFromConfig(conf.Config)
	hostedZones, err := route53Client.ListHostedZonesByName(
		context.Background(),
		&route53.ListHostedZonesByNameInput{
			DNSName: &hostedZoneName,
		},
	)
	if err != nil {
		return "", fmt.Errorf("error listing hosted zones: %w", err)
	}

	var hostedZoneID string

	for _, zone := range hostedZones.HostedZones {
		if *zone.Name == fmt.Sprintf(`%s%s`, hostedZoneName, ".") {
			hostedZoneID = strings.Split(*zone.Id, "/")[2]
		}
	}

	if hostedZoneID == "" {
		return "", fmt.Errorf("error finding hosted zone ID for hosted zone %s", hostedZoneName)
	}

	return hostedZoneID, nil
}

// GetHostedZone returns an object detailing a hosted zone
func (conf *Configuration) GetHostedZone(hostedZoneID string) (*route53.GetHostedZoneOutput, error) {
	route53Client := route53.NewFromConfig(conf.Config)
	hostedZone, err := route53Client.GetHostedZone(context.Background(), &route53.GetHostedZoneInput{
		Id: aws.String(hostedZoneID),
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching details for hosted zone %s: %s", hostedZoneID, err)
	}

	return hostedZone, nil
}

// GetHostedZone returns an object detailing a hosted zone
func (conf *Configuration) GetHostedZones() ([]string, error) {
	route53Client := route53.NewFromConfig(conf.Config)
	hostedZones, err := route53Client.ListHostedZones(context.Background(), &route53.ListHostedZonesInput{})
	if err != nil {
		return nil, fmt.Errorf("error listing hosted zones: %w", err)
	}

	domainList := make([]string, 0, len(hostedZones.HostedZones))
	for _, domain := range hostedZones.HostedZones {
		domainList = append(domainList, *domain.Name)
	}

	return domainList, nil
}

// GetHostedZoneNameServers returns nameservers for a hosted zone if available
// for private zones, nothing is returned
func (conf *Configuration) GetHostedZoneNameServers(domainName string) (bool, []string, error) {
	hostedZoneID, err := conf.GetHostedZoneID(domainName)
	if err != nil {
		return false, nil, err
	}

	hostedZone, err := conf.GetHostedZone(hostedZoneID)
	if err != nil {
		return false, nil, err
	}

	switch hostedZone.HostedZone.Config.PrivateZone {
	case true:
		return true, nil, nil
	case false:
		return false, hostedZone.DelegationSet.NameServers, nil
	}

	return false, nil, nil
}
