// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package auth

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// TokenProvider is the interface that wraps an authentication mechanism and
// allows to obtain tokens.
type TokenProvider interface {
	// Token returns a valid OAuth token, or an error.
	Token(ctx context.Context) (string, error)
}

// credentialTokenProvider extends azidentity.ClientSecretCredential with the
// the TokenProvider interface.
type credentialTokenProvider azidentity.ClientSecretCredential

// Token returns an oauth token that can be used for bearer authorization.
func (provider *credentialTokenProvider) Token(ctx context.Context) (string, error) {
	inner := (*azidentity.ClientSecretCredential)(provider)
	tk, err := inner.GetToken(
		ctx, policy.TokenRequestOptions{Scopes: []string{"https://manage.office.com/.default"}},
	)
	if err != nil {
		return "", err
	}
	return tk.Token, nil
}
