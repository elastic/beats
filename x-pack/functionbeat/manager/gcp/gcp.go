// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import (
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/x-pack/functionbeat/function/provider"
	"github.com/elastic/beats/v7/x-pack/functionbeat/provider/gcp/gcp"
)

// Bundle exposes the trigger supported by the gcp provider.
var Bundle = provider.MustCreate(
	"gcp",
	provider.NewDefaultProvider("gcp", NewCLI, NewTemplateBuilder),
	feature.NewDetails("Google Cloud Functions", "listen to events on Google Cloud", feature.Stable),
).MustAddFunction("pubsub",
	gcp.NewPubSub,
	gcp.PubSubDetails(),
).MustAddFunction("storage",
	gcp.NewStorage,
	gcp.StorageDetails(),
).Bundle()
