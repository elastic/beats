// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package providers

import (
	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/beats/v7/x-pack/libbeat/autodiscover/providers/aws/ec2"
	"github.com/elastic/beats/v7/x-pack/libbeat/autodiscover/providers/aws/elb"
	"github.com/elastic/beats/v7/x-pack/libbeat/autodiscover/providers/nomad"
)

var KnownProviders = map[string]autodiscover.ProviderBuilder{
	nomad.ProviderName: nomad.AutodiscoverBuilder,
	ec2.ProviderName:   ec2.AutodiscoverBuilder,
	elb.ProviderName:   elb.AutodiscoverBuilder,
}
