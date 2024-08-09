/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package google

import (
	"fmt"
	"net/http"
	"time"

	compute "cloud.google.com/go/compute/apiv1"
	computepb "cloud.google.com/go/compute/apiv1/computepb"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// GetRegions lists all available regions
func (conf *Configuration) GetRegions() ([]string, error) {
	var regionList []string

	creds, err := google.CredentialsFromJSON(conf.Context, []byte(conf.KeyFile), secretmanager.DefaultAuthScopes()...)
	if err != nil {
		return nil, fmt.Errorf("could not create google storage client credentials: %w", err)
	}

	client, err := compute.NewRegionsRESTClient(conf.Context, option.WithCredentials(creds))
	if err != nil {
		return []string{}, fmt.Errorf("could not create google compute client: %w", err)
	}
	defer client.Close()

	req := &computepb.ListRegionsRequest{
		Project: conf.Project,
	}

	it := client.List(conf.Context, req)
	for {
		pair, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return []string{}, err
		}
		regionList = append(regionList, pair.GetName())
	}

	return regionList, nil
}

func (conf *Configuration) GetZones() ([]string, error) {
	var zoneList []string

	creds, err := google.CredentialsFromJSON(conf.Context, []byte(conf.KeyFile), secretmanager.DefaultAuthScopes()...)
	if err != nil {
		return nil, fmt.Errorf("could not create google storage client credentials: %w", err)
	}

	client, err := compute.NewZonesRESTClient(conf.Context, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("could not create google compute client: %w", err)
	}
	defer client.Close()

	req := &computepb.ListZonesRequest{
		Project: conf.Project,
	}

	it := client.List(conf.Context, req)
	for {
		pair, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return []string{}, err
		}
		zoneList = append(zoneList, pair.GetName())
	}

	return zoneList, nil
}

// GetDomainApexContent determines whether or not a target domain features
// a host responding at zone apex
func GetDomainApexContent(domainName string) bool {
	timeout := time.Duration(5 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}

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
