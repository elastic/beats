// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

// Shared types are defined here in this package to make structuring better
package types

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

// Source , it is the cursor source
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
	SharedKeyType        string = "sharedKeyType"
	ConnectionStringType string = "connectionStringType"
	Json                 string = "application/json"
)

// currently only shared key & connection string types of credentials are supported
type ServiceCredentials struct {
	SharedKeyCreds     *azblob.SharedKeyCredential
	ConnectionStrCreds string
	Ctype              string
}

type BlobCredentials struct {
	ServiceCreds  *ServiceCredentials
	BlobName      string
	ContainerName string
}

var AllowedContentTypes = map[string]bool{
	Json: true,
}
