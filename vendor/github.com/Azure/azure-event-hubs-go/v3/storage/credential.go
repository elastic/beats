package storage

//	MIT License
//
//	Copyright (c) Microsoft Corporation. All rights reserved.
//
//	Permission is hereby granted, free of charge, to any person obtaining a copy
//	of this software and associated documentation files (the "Software"), to deal
//	in the Software without restriction, including without limitation the rights
//	to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
//	copies of the Software, and to permit persons to whom the Software is
//	furnished to do so, subject to the following conditions:
//
//	The above copyright notice and this permission notice shall be included in all
//	copies or substantial portions of the Software.
//
//	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
//	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
//	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
//	AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
//	LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
//	OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
//	SOFTWARE

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/Azure/azure-amqp-common-go/v3/aad"
	"github.com/Azure/azure-pipeline-go/pipeline"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2017-10-01/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/date"
)

type (
	// AADSASCredential represents a token provider for Azure Storage SAS using AAD to authorize signing
	AADSASCredential struct {
		azblob.Credential
		ResourceGroup    string
		SubscriptionID   string
		AccountName      string
		ContainerName    string
		aadTokenProvider *adal.ServicePrincipalToken
		token            *SASToken
		env              *azure.Environment
		lockMu           sync.Mutex
	}

	// SASToken contains the expiry time and token for a given SAS
	SASToken struct {
		expiry time.Time
		sas    string
	}

	// AADSASCredentialOption provides options for configuring AAD SAS Token Providers
	AADSASCredentialOption func(*aad.TokenProviderConfiguration) error
)

// AADSASCredentialWithEnvironmentVars configures the TokenProvider using the environment variables available
//
// 1. Client Credentials: attempt to authenticate with a Service Principal via "AZURE_TENANT_ID", "AZURE_CLIENT_ID" and
//    "AZURE_CLIENT_SECRET"
//
// 2. Client Certificate: attempt to authenticate with a Service Principal via "AZURE_TENANT_ID", "AZURE_CLIENT_ID",
//    "AZURE_CERTIFICATE_PATH" and "AZURE_CERTIFICATE_PASSWORD"
//
// 3. Managed Service Identity (MSI): attempt to authenticate via MSI
//
//
// The Azure Environment used can be specified using the name of the Azure Environment set in "AZURE_ENVIRONMENT" var.
func AADSASCredentialWithEnvironmentVars() AADSASCredentialOption {
	return func(config *aad.TokenProviderConfiguration) error {
		config.TenantID = os.Getenv("AZURE_TENANT_ID")
		config.ClientID = os.Getenv("AZURE_CLIENT_ID")
		config.ClientSecret = os.Getenv("AZURE_CLIENT_SECRET")
		config.CertificatePath = os.Getenv("AZURE_CERTIFICATE_PATH")
		config.CertificatePassword = os.Getenv("AZURE_CERTIFICATE_PASSWORD")

		if config.Env == nil {
			env, err := azureEnvFromEnvironment()
			if err != nil {
				return err
			}
			config.Env = env
		}
		return nil
	}
}

// NewAADSASCredential constructs a SAS token provider for Azure storage using Azure Active Directory credentials
//
// canonicalizedResource should be formed as described here: https://docs.microsoft.com/en-us/rest/api/storagerp/storageaccounts/listservicesas
func NewAADSASCredential(subscriptionID, resourceGroup, accountName, containerName string, opts ...AADSASCredentialOption) (*AADSASCredential, error) {
	config := &aad.TokenProviderConfiguration{
		ResourceURI: azure.PublicCloud.ResourceManagerEndpoint,
		Env:         &azure.PublicCloud,
	}

	for _, opt := range opts {
		err := opt(config)
		if err != nil {
			return nil, err
		}
	}

	spToken, err := config.NewServicePrincipalToken()
	if err != nil {
		return nil, err
	}

	return &AADSASCredential{
		aadTokenProvider: spToken,
		env:              config.Env,
		SubscriptionID:   subscriptionID,
		ResourceGroup:    resourceGroup,
		AccountName:      accountName,
		ContainerName:    containerName,
	}, nil
}

// New creates a credential policy object.
func (cred *AADSASCredential) New(next pipeline.Policy, po *pipeline.PolicyOptions) pipeline.Policy {
	return pipeline.PolicyFunc(func(ctx context.Context, request pipeline.Request) (pipeline.Response, error) {
		// Add a x-ms-date header if it doesn't already exist
		token, err := cred.getToken(ctx)
		if err != nil {
			return nil, err
		}

		if request.URL.RawQuery != "" {
			request.URL.RawQuery = request.URL.RawQuery + "&" + token.sas
		} else {
			request.URL.RawQuery = token.sas
		}

		response, err := next.Do(ctx, request)
		return response, err
	})
}

// GetToken fetches a Azure Storage SAS token using an AAD token
func (cred *AADSASCredential) getToken(ctx context.Context) (SASToken, error) {
	cred.lockMu.Lock()
	defer cred.lockMu.Unlock()

	span, ctx := startConsumerSpanFromContext(ctx, "storage.AADSASCredential.getToken")
	defer span.End()

	if cred.token != nil {
		if !cred.token.expiry.Before(time.Now().Add(5 * time.Minute)) {
			return *cred.token, nil
		}
	}
	token, err := cred.refreshToken(ctx, "/blob/"+cred.AccountName+"/"+cred.ContainerName)
	if err != nil {
		return SASToken{}, err
	}

	cred.token = &token
	return token, nil
}

func (cred *AADSASCredential) refreshToken(ctx context.Context, canonicalizedResource string) (SASToken, error) {
	span, ctx := startConsumerSpanFromContext(ctx, "storage.AADSASCredential.refreshToken")
	defer span.End()

	now := time.Now().Add(-1 * time.Second)
	expiry := now.Add(1 * time.Hour)
	client := storage.NewAccountsClientWithBaseURI(cred.env.ResourceManagerEndpoint, cred.SubscriptionID)
	client.Authorizer = autorest.NewBearerAuthorizer(cred.aadTokenProvider)
	res, err := client.ListAccountSAS(ctx, cred.ResourceGroup, cred.AccountName, storage.AccountSasParameters{
		Protocols:              storage.HTTPS,
		ResourceTypes:          storage.SignedResourceTypesS + storage.SignedResourceTypesC + storage.SignedResourceTypesO,
		Services:               storage.B,
		SharedAccessStartTime:  &date.Time{Time: now.Round(time.Second).UTC()},
		SharedAccessExpiryTime: &date.Time{Time: expiry.Round(time.Second).UTC()},
		Permissions:            storage.R + storage.W + storage.D + storage.L + storage.A + storage.C + storage.U,
	})

	if err != nil {
		return SASToken{}, err
	}

	return SASToken{
		sas:    *res.AccountSasToken,
		expiry: expiry,
	}, err
}

func azureEnvFromEnvironment() (*azure.Environment, error) {
	envName := os.Getenv("AZURE_ENVIRONMENT")

	var env azure.Environment
	if envName == "" {
		env = azure.PublicCloud
	} else {
		var err error
		env, err = azure.EnvironmentFromName(envName)
		if err != nil {
			return nil, err
		}
	}
	return &env, nil
}
