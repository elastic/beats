// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"fmt"
	"regexp"
	"time"
	"unicode"

	humanize "github.com/dustin/go-humanize"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
)

// Config expose the configuration option the AWS provider.
type Config struct {
	Endpoint     string `config:"endpoint" validate:"nonzero,required"`
	DeployBucket bucket `config:"deploy_bucket" validate:"nonzero,required"`
}

// maxMegabytes maximums memory that a lambda can use.
const maxMegabytes = 3008

// DefaultLambdaConfig confguration for AWS lambda function.
var (
	DefaultLambdaConfig = &lambdaConfig{
		MemorySize:  128 * 1024 * 1024,
		Timeout:     time.Second * 3,
		Concurrency: 5,
	}

	// Source: https://docs.aws.amazon.com/lambda/latest/dg/API_CreateFunction.html#SSS-CreateFunction-request-Role
	arnRolePattern = "arn:(aws[a-zA-Z-]*)?:iam::\\d{12}:role/?[a-zA-Z_0-9+=,.@\\-_/]+"
	roleRE         = regexp.MustCompile(arnRolePattern)
)

type lambdaConfig struct {
	Concurrency      int               `config:"concurrency" validate:"min=0,max=1000"`
	DeadLetterConfig *deadLetterConfig `config:"dead_letter_config"`
	Description      string            `config:"description"`
	MemorySize       MemSizeFactor64   `config:"memory_size"`
	Timeout          time.Duration     `config:"timeout" validate:"nonzero,positive"`
	Role             string            `config:"role"`
	VPCConfig        *vpcConfig        `config:"virtual_private_cloud"`
}

func (c *lambdaConfig) Validate() error {
	if c.MemorySize.Megabytes() == 0 {
		return fmt.Errorf("'memory_size' need to be higher than 0 and must be a factor 64")
	}

	if c.MemorySize.Megabytes() > maxMegabytes {
		return fmt.Errorf("'memory_size' must be lower than %d", maxMegabytes)
	}

	if c.Role != "" && !roleRE.MatchString(c.Role) {
		return fmt.Errorf("invalid role: '%s', name must match pattern %s", c.Role, arnRolePattern)
	}

	return nil
}

type deadLetterConfig struct {
	TargetArn string `config:"target_arn"`
}

type vpcConfig struct {
	SecurityGroupIDs []string `config:"security_group_ids" validate:"required"`
	SubnetIDs        []string `config:"subnet_ids" validate:"required"`
}

// MemSizeFactor64 implements a human understandable format for bytes but also make sure that all
// values used are a factory of 64.
type MemSizeFactor64 int

// Unpack converts a size defined from a human readable format into bytes and ensure that the value
// is a factoru of 64.
func (m *MemSizeFactor64) Unpack(v string) error {
	sz, err := humanize.ParseBytes(v)
	if isRawBytes(v) {
		cfgwarn.Deprecate("7.0", "size now requires a unit (KiB, MiB, etc...), current value: %s.", v)
	}
	if err != nil {
		return err
	}

	if sz%64 != 0 {
		return fmt.Errorf("number is not a factor of 64, %d bytes (user value: %s)", sz, v)
	}

	*m = MemSizeFactor64(sz)
	return nil
}

// Megabytes return the value in megatebytes.
func (m *MemSizeFactor64) Megabytes() int {
	return int(*m) / 1024 / 1024
}

func isRawBytes(v string) bool {
	for _, c := range v {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

type bucket string

// Do some high level validation on the bucket name, they have strict validations on the name on the API side.
// DOC: https://docs.aws.amazon.com/AmazonS3/latest/dev/BucketRestrictions.html#bucketnamingrules
func (b *bucket) Unpack(s string) error {
	const max = 63
	const min = 3
	if len(s) > max {
		return fmt.Errorf("bucket name '%s' is too long, name are restricted to %d chars", s, max)
	}

	if len(s) < min {
		return fmt.Errorf("bucket name '%s' is too short, name need to be at least %d chars long", s, min)
	}

	*b = bucket(s)
	return nil
}
