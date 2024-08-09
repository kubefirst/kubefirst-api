/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/rs/zerolog/log"
)

// ElbTags describes a pair of tags assigned to an Elastic Load Balancer
type ElbTags struct {
	Key   string
	Value string
}

// ElbDeletionParameters describes an Elastic Load Balancer name and source
// security group to delete
type ElbDeletionParameters struct {
	ElbName                 string
	ElbSourceSecurityGroups []string
}

// GetLoadBalancersForDeletion gets all load balancers and returns details for
// a load balancer associated with the target EKS cluster
func (conf *Configuration) GetLoadBalancersForDeletion(eksClusterName string) ([]ElbDeletionParameters, error) {
	elbClient := elasticloadbalancing.NewFromConfig(conf.Config)

	// Get all elastic load balancers
	elbs, err := elbClient.DescribeLoadBalancers(context.Background(), &elasticloadbalancing.DescribeLoadBalancersInput{})
	if err != nil {
		return nil, fmt.Errorf("error getting load balancers: %w", err)
	}

	// Build list of Elastic Load Balancer names
	elbNames := make([]string, 0, len(elbs.LoadBalancerDescriptions))
	for _, lb := range elbs.LoadBalancerDescriptions {
		elbNames = append(elbNames, *lb.LoadBalancerName)
	}

	// Get tags for each Elastic Load Balancer
	elbTags := make(map[string][]ElbTags)
	for _, elb := range elbNames {
		// Describe tags per Elastic Load Balancer
		tags, err := elbClient.DescribeTags(context.Background(), &elasticloadbalancing.DescribeTagsInput{
			LoadBalancerNames: []string{elb},
		})
		if err != nil {
			return nil, fmt.Errorf("error getting tags for load balancer %q: %w", elb, err)
		}

		// Compile tags
		tagsContainer := make([]ElbTags, 0)
		for _, tag := range tags.TagDescriptions {
			for _, desc := range tag.Tags {
				tagsContainer = append(tagsContainer, ElbTags{Key: *desc.Key, Value: *desc.Value})
			}
		}

		// Add to map
		elbTags[elb] = tagsContainer
	}

	// Return matched load balancers
	elbsToDelete := []ElbDeletionParameters{}
	for key, value := range elbTags {
		for _, tag := range value {
			if tag.Key == fmt.Sprintf("kubernetes.io/cluster/%s", eksClusterName) && tag.Value == "owned" {
				elb, err := elbClient.DescribeLoadBalancers(context.Background(), &elasticloadbalancing.DescribeLoadBalancersInput{
					LoadBalancerNames: []string{key},
				})
				if err != nil {
					return nil, fmt.Errorf("error getting load balancer details for %q: %w", key, err)
				}

				if len(elb.LoadBalancerDescriptions) == 0 {
					continue
				}

				targetSecurityGroups := elb.LoadBalancerDescriptions[0].SecurityGroups
				elbsToDelete = append(elbsToDelete, ElbDeletionParameters{
					ElbName:                 key,
					ElbSourceSecurityGroups: targetSecurityGroups,
				})
			}
		}
	}

	return elbsToDelete, nil
}

// DeleteEKSSecurityGroups deletes security groups associated with an EKS cluster
func (conf *Configuration) DeleteEKSSecurityGroups(region string, clusterName string) error {
	ec2Client := ec2.NewFromConfig(conf.Config, func(o *ec2.Options) {
		o.Region = region
	})

	// Get dependent security groups
	filterName := "tag-key"
	maxResults := int32(1000)
	secGroups, err := ec2Client.DescribeSecurityGroups(context.Background(), &ec2.DescribeSecurityGroupsInput{
		MaxResults: &maxResults,
		Filters: []ec2Types.Filter{{
			Name:   &filterName,
			Values: []string{fmt.Sprintf("kubernetes.io/cluster/%s", clusterName)},
		}},
	})
	if err != nil {
		return fmt.Errorf("error getting security groups for cluster %q: %w", clusterName, err)
	}

	// Delete matched security groups
	for _, sg := range secGroups.SecurityGroups {
		log.Info().Msgf("revoking ingress rule %v", sg)
		_, err := ec2Client.RevokeSecurityGroupIngress(context.Background(), &ec2.RevokeSecurityGroupIngressInput{
			GroupName:     sg.GroupName,
			IpPermissions: sg.IpPermissions,
		})
		if err != nil {
			return fmt.Errorf("error revoking ingress rule: %w", err)
		}
		log.Info().Msgf("revoked ingress rule %v", sg)

		log.Info().Msgf("revoking egress rule %v", sg)
		_, err = ec2Client.RevokeSecurityGroupEgress(context.Background(), &ec2.RevokeSecurityGroupEgressInput{
			GroupId:       sg.GroupId,
			IpPermissions: sg.IpPermissions,
		})
		if err != nil {
			return fmt.Errorf("error revoking egress rule: %w", err)
		}
		log.Info().Msgf("revoked egress rule %v", sg)
	}

	// Delete matched security groups
	for _, sg := range secGroups.SecurityGroups {
		log.Info().Msgf("preparing to delete eks security group %s / %s", *sg.GroupName, *sg.GroupId)

		_, err = ec2Client.DeleteSecurityGroup(context.Background(), &ec2.DeleteSecurityGroupInput{
			GroupId: sg.GroupId,
		})
		if err != nil {
			return fmt.Errorf("error deleting security group %s / %s: %w", *sg.GroupName, *sg.GroupId, err)
		}

		log.Info().Msgf("deleted security group %s / %s", *sg.GroupName, *sg.GroupId)
	}

	return nil
}

// DeleteSecurityGroup deletes a security group
func (conf *Configuration) DeleteSecurityGroup(region string, sgID string) error {
	ec2Client := ec2.NewFromConfig(conf.Config, func(o *ec2.Options) {
		o.Region = region
	})

	// Get dependent security groups
	filterName := "group-id"
	dependentSecurityGroups, err := ec2Client.DescribeSecurityGroups(context.Background(), &ec2.DescribeSecurityGroupsInput{
		Filters: []ec2Types.Filter{{
			Name:   &filterName,
			Values: []string{sgID},
		}},
	})
	if err != nil {
		return fmt.Errorf("error getting security groups for cluster %q: %w", sgID, err)
	}

	// Delete rules
	for _, sg := range dependentSecurityGroups.SecurityGroups {
		log.Info().Msgf("revoking ingress rule: %v", sg)
		_, err := ec2Client.RevokeSecurityGroupIngress(context.Background(), &ec2.RevokeSecurityGroupIngressInput{
			GroupName:     sg.GroupName,
			IpPermissions: sg.IpPermissions,
		})
		if err != nil {
			return fmt.Errorf("error during rule removal: %w", err)
		}

		log.Info().Msgf("revoking egress rule: %v", sg)
		_, err = ec2Client.RevokeSecurityGroupEgress(context.Background(), &ec2.RevokeSecurityGroupEgressInput{
			GroupId:       sg.GroupId,
			IpPermissions: sg.IpPermissions,
		})
		if err != nil {
			return fmt.Errorf("error during rule removal: %w", err)
		}

		log.Info().Msgf("preparing to delete eks security group %s / %s", *sg.GroupName, *sg.GroupId)
		_, err = ec2Client.DeleteSecurityGroup(context.Background(), &ec2.DeleteSecurityGroupInput{
			GroupId: sg.GroupId,
		})
		if err != nil {
			return fmt.Errorf("error deleting security group %s / %s: %w", *sg.GroupName, *sg.GroupId, err)
		}

		log.Info().Msgf("deleted security group %s / %s", *sg.GroupName, *sg.GroupId)
	}

	return nil
}

// DeleteElasticLoadBalancer deletes an Elastic Load Balancer associated with an EKS cluster
func (conf *Configuration) DeleteElasticLoadBalancer(elbdp ElbDeletionParameters) error {
	elbClient := elasticloadbalancing.NewFromConfig(conf.Config)

	_, err := elbClient.DeleteLoadBalancer(context.Background(), &elasticloadbalancing.DeleteLoadBalancerInput{
		LoadBalancerName: &elbdp.ElbName,
	})
	if err != nil {
		return fmt.Errorf("error deleting elastic load balancer %s: %w", elbdp.ElbName, err)
	}

	log.Info().Msgf("deleted elastic load balancer %s", elbdp.ElbName)

	return nil
}
