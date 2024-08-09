/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
)

// Configuration stores session data to organize all AWS functions into a single struct
type Configuration struct {
	Config aws.Config
}

type Route53AlterResourceRecord struct {
	hostedZoneName    string
	hostedZoneID      string
	route53RecordName string
	input             *route53.ChangeResourceRecordSetsInput
}
