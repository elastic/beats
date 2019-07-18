// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/elastic/beats/x-pack/functionbeat/config"
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
	DefaultLambdaConfig = &LambdaConfig{
		MemorySize:  128 * 1024 * 1024,
		Timeout:     time.Second * 3,
		Concurrency: 5,
	}

	// Source: https://docs.aws.amazon.com/lambda/latest/dg/API_CreateFunction.html#SSS-CreateFunction-request-Role
	arnRolePattern = "arn:(aws[a-zA-Z-]*)?:iam::\\d{12}:role/?[a-zA-Z_0-9+=,.@\\-_/]+"
	roleRE         = regexp.MustCompile(arnRolePattern)

	// Chars for resource name anything else will be replaced.
	validChars = regexp.MustCompile("[^a-zA-Z0-9]")
)

// LambdaConfig stores the common configuration of Lambda functions.
type LambdaConfig struct {
	Concurrency      int                    `config:"concurrency" validate:"min=0,max=1000"`
	DeadLetterConfig *deadLetterConfig      `config:"dead_letter_config"`
	Description      string                 `config:"description"`
	MemorySize       config.MemSizeFactor64 `config:"memory_size"`
	Timeout          time.Duration          `config:"timeout" validate:"nonzero,positive"`
	Role             string                 `config:"role"`
	VPCConfig        *vpcConfig             `config:"virtual_private_cloud"`
	Tags             map[string]string      `config:"tags"`
}

// Validate checks a LambdaConfig
func (c *LambdaConfig) Validate() error {
	if c.MemorySize.Megabytes() == 0 {
		return fmt.Errorf("'memory_size' need to be higher than 0 and must be a factor 64")
	}

	if c.MemorySize.Megabytes() > maxMegabytes {
		return fmt.Errorf("'memory_size' must be lower than %d", maxMegabytes)
	}

	if c.Role != "" && !roleRE.MatchString(c.Role) {
		return fmt.Errorf("invalid role: '%s', name must match pattern %s", c.Role, arnRolePattern)
	}

	return validateTags(c.Tags)
}

type deadLetterConfig struct {
	TargetArn string `config:"target_arn"`
}

type vpcConfig struct {
	SecurityGroupIDs []string `config:"security_group_ids" validate:"required"`
	SubnetIDs        []string `config:"subnet_ids" validate:"required"`
}

func validateTags(tags map[string]string) error {
	for key, val := range tags {
		if strings.HasPrefix(key, "aws:") {
			return fmt.Errorf("key '%s' cannot be prefixed with 'aws:'", key)
		}
		if strings.HasPrefix(val, "aws:") {
			return fmt.Errorf("value '%s' cannot be prefixed with 'aws:'", val)
		}
		keyLen := utf8.RuneCountInString(key)
		if keyLen > 127 {
			return fmt.Errorf("too long key; expected 127 chars, but got %d", keyLen)
		}
		valLen := utf8.RuneCountInString(val)
		if valLen > 255 {
			return fmt.Errorf("too long value; expected 255 chars, but got %d", valLen)
		}
	}

	return nil
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

// NormalizeResourceName extracts invalid chars.
func NormalizeResourceName(s string) string {
	return validChars.ReplaceAllString(s, "")
}
