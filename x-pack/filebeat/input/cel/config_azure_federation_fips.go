// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build requirefips

package cel

import (
	"context"
	"errors"
	"net/http"

	"github.com/elastic/beats/v7/pkg/logp"
)

// Azure identity federation needs the Azure SDK, which is excluded from FIPS builds.
const azureCloudConnectorsSupported = false

func (o *oAuth2Config) azureCloudConnectorsClient(context.Context, *http.Client, *logp.Logger) (*http.Client, error) {
	return nil, errors.New("azure.use_cloud_connectors is not supported in FIPS builds")
}
