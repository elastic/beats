// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	humanize "github.com/dustin/go-humanize"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
)

// maxMegabytes maximums memory that a lambda can use.
const maxMegabytes = 3008

// DefaultConfig confguration for AWS lambda function.
var DefaultConfig = lambdaConfig{
	MemorySize: 128 * 1024 * 1024,
	Timeout:    time.Second * 3,
}

type lambdaConfig struct {
	DeadLetterConfig *deadLetterConfig `config:"dead_letter_config"`
	Description      string            `config:"description"`
	KMSKeyArn        string            `config:"kms_key_arn"`
	MemorySize       MemSizeFactor64   `config:"memory_size"`
	Tags             map[string]string `config:"tags"`
	Timeout          time.Duration     `config:"timeout" validate:"nonzero,positive"`
	Tracing          tracingConfig     `config:"tracing"`
	VPC              vpcConfig         `config:"vpc"`
}

func (c *lambdaConfig) Validate() error {
	if c.MemorySize.Megabytes() == 0 {
		return fmt.Errorf("'memory_size' need to be higher than 0 and must be a factor 64")
	}

	if c.MemorySize.Megabytes() > int64(maxMegabytes) {
		return fmt.Errorf("'memory_size' must be lower than %d", maxMegabytes)
	}

	return nil
}

type vpcConfig struct {
	SecurityGroupIds []string `config:"security_group_ids"`
	SubnetIds        []string `config:"subnet_ids"`
}

type tracingConfig lambda.TracingMode

var tracingConfigMappings = map[string]string{
	"active":      "Active",
	"passthrough": "PassThrough",
}

func (m *tracingConfig) Unpack(v string) error {
	v = strings.ToLower(v)
	option, ok := tracingConfigMappings[v]
	if !ok {
		return fmt.Errorf(
			"unknown value for tracing config, received: '%s', valid values are: 'active' or 'passthrough'",
			v,
		)
	}
	*m = tracingConfig(option)
	return nil
}

type deadLetterConfig struct {
	TargetArn string `config:"target_arn"`
}

// MemSizeFactor64 implements a human understandable format for bytes but also make sure that all
// values used are a factory of 64.
type MemSizeFactor64 int64

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
func (m *MemSizeFactor64) Megabytes() int64 {
	return int64(*m) / 1024 / 1024
}

func isRawBytes(v string) bool {
	for _, c := range v {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}
