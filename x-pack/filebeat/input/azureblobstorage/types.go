// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureblobstorage

import "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

const (
	inputName                  string = "azureblobstorage"
	sharedKeyCredential        string = "sharedKeyCredential"
	connectionStringCredential string = "connectionStringCredential"
)

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
	"application/json": true,
}
