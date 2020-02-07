// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/ec2iface"
	"github.com/pkg/errors"
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
func GetRegions(svc ec2iface.ClientAPI) (completeRegionsList []string, err error) {
	input := &ec2.DescribeRegionsInput{}
	req := svc.DescribeRegionsRequest(input)
	output, err := req.Send(context.TODO())
	if err != nil {
		err = errors.Wrap(err, "Failed DescribeRegions")
		return
	}
	for _, region := range output.Regions {
		completeRegionsList = append(completeRegionsList, *region.RegionName)
	}
	return
}
