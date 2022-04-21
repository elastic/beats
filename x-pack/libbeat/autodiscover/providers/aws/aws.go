// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// SafeString makes handling AWS *string types easier.
// The AWS lib never returns plain strings, always using pointers, probably for memory efficiency reasons.
// This is a bit odd, because strings are just pointers into byte arrays, however this is the choice they've made.
// This will return the plain version of the given string or an empty string if the pointer is null
func SafeString(str *string) string {
	if str == nil {
		return ""
	}

	return *str
}

// GetRegions makes DescribeRegions API call to list all regions from AWS
func GetRegions(svc *ec2.Client) ([]string, error) {
	output, err := svc.DescribeRegions(context.TODO(), &ec2.DescribeRegionsInput{})
	if err != nil {
		return nil, fmt.Errorf("Failed DescribeRegions: %w", err)
	}

	completeRegionsList := make([]string, 0)
	for _, region := range output.Regions {
		completeRegionsList = append(completeRegionsList, *region.RegionName)
	}

	return completeRegionsList, nil
}
