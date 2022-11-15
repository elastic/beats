// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Shared types are defined here to make structuring better
package azureblobstorage

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

// Source, it is the cursor source
type Source struct {
	ContainerName string
	AccountName   string
	MaxWorkers    int
	Poll          bool
	PollInterval  time.Duration
}

func (s *Source) Name() string {
	return s.AccountName + "::" + s.ContainerName
}

const (
	sharedKeyType        string = "sharedKeyType"
	connectionStringType string = "connectionStringType"
	json                 string = "application/json"
)

// currently only shared key & connection string types of credentials are supported
type serviceCredentials struct {
	sharedKeyCreds     *azblob.SharedKeyCredential
	connectionStrCreds string
	cType              string
}

type blobCredentials struct {
	serviceCreds  *serviceCredentials
	blobName      string
	containerName string
}

var allowedContentTypes = map[string]bool{
	json: true,
}
