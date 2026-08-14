// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

// Shared types are defined here to make structuring better
package azureblobstorage

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

// Source, it is the cursor source
type Source struct {
	ContainerName            string
	AccountName              string
	BatchSize                int
	MaxWorkers               int
	Poll                     bool
	PollInterval             time.Duration
	TimeStampEpoch           *int64
	FileSelectors            []fileSelectorConfig
	ReaderConfig             readerConfig
	ExpandEventListFromField string
	PathPrefix               string
	Retry                    retryConfig
}

func (s *Source) Name() string {
	return s.AccountName + "::" + s.ContainerName
}

const (
	sharedKeyType        = "sharedKeyType"
	connectionStringType = "connectionStringType"
	oauth2Type           = "oauth2Type"
	managedIdentityType  = "managedIdentityType"
	jsonType             = "application/json"
	octetType            = "application/octet-stream"
	ndJsonType           = "application/x-ndjson"
	gzType               = "application/x-gzip"
	csvType              = "text/csv"
	encodingGzip         = "gzip"
)

// serviceCredentials holds the credential that the input uses for every request
// to the storage account, and the client options that go with it.
type serviceCredentials struct {
	// tokenCreds is set for the oauth2 and managed identity types.
	tokenCreds         azcore.TokenCredential
	sharedKeyCreds     *azblob.SharedKeyCredential
	connectionStrCreds string

	// cType names which of the credentials above is set.
	cType string
	// clientOpts are the resolved options for storage clients, including the retry
	// policy. Blob clients reuse them.
	clientOpts azcore.ClientOptions
}

type blobCredentials struct {
	serviceCreds  *serviceCredentials
	blobName      string
	containerName string
}

var allowedContentTypes = map[string]bool{
	jsonType:   true,
	octetType:  true,
	ndJsonType: true,
	gzType:     true,
	csvType:    true,
}
