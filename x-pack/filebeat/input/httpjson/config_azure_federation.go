// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package httpjson

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"golang.org/x/oauth2"

	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/identityfederation"
)

const azureCloudConnectorsSupported = true

// azureCloudConnectorsClient returns a client that authenticates requests with
// Azure access tokens from the cloud connector identity. Token requests to Entra
// use client, so they go through the configured proxy and TLS settings.
func (o *oAuth2Config) azureCloudConnectorsClient(ctx context.Context, client *http.Client, log *logp.Logger) (*http.Client, error) {
	options := &azidentity.ClientAssertionCredentialOptions{
		ClientOptions: azcore.ClientOptions{Transport: client},
	}
	tokenSource, err := identityfederation.AzureNewCloudConnectorTokenSource(ctx, o.AzureTenantID, o.ClientID, o.azureScopes(), options, log)
	if err != nil {
		return nil, fmt.Errorf("oauth2 client: error creating Azure identity federation credentials: %w", err)
	}
	return oauth2.NewClient(ctx, tokenSource), nil
}
