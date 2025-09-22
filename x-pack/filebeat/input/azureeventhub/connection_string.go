// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This code is a modified version of the code from the Azure SDK for Go.
//
// The original code is available at:
// https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/messaging/azeventhubs/internal/exported/connection_string_properties.go

//go:build !aix

package azureeventhub

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ConnectionStringProperties are the properties of a connection string
// as returned by [ParseConnectionString].
type ConnectionStringProperties struct {
	// Endpoint is the Endpoint value in the connection string.
	// Ex: sb://example.servicebus.windows.net
	Endpoint string

	// EntityPath is EntityPath value in the connection string.
	EntityPath *string

	// FullyQualifiedNamespace is the Endpoint value without the protocol scheme.
	// Ex: example.servicebus.windows.net
	FullyQualifiedNamespace string

	// SharedAccessKey is the SharedAccessKey value in the connection string.
	SharedAccessKey *string

	// SharedAccessKeyName is the SharedAccessKeyName value in the connection string.
	SharedAccessKeyName *string

	// SharedAccessSignature is the SharedAccessSignature value in the connection string.
	SharedAccessSignature *string

	// Emulator indicates that the connection string is for an emulator:
	// ex: Endpoint=localhost:6765;SharedAccessKeyName=<< REDACTED >>;SharedAccessKey=<< REDACTED >>;UseDevelopmentEmulator=true
	Emulator bool
}

// ParseConnectionString takes a connection string from the Azure portal and returns the
// parsed representation.
//
// There are two supported formats:
//
//  1. Connection strings generated from the portal (or elsewhere) that contain an embedded key and keyname.
//
//  2. A connection string with an embedded SharedAccessSignature:
//     Endpoint=sb://<sb>.servicebus.windows.net;SharedAccessSignature=SharedAccessSignature sr=<sb>.servicebus.windows.net&sig=<base64-sig>&se=<expiry>&skn=<keyname>"
func parseConnectionString(connStr string) (ConnectionStringProperties, error) {
	const (
		endpointKey              = "Endpoint"
		sharedAccessKeyNameKey   = "SharedAccessKeyName"
		sharedAccessKeyKey       = "SharedAccessKey"
		entityPathKey            = "EntityPath"
		sharedAccessSignatureKey = "SharedAccessSignature"
		useEmulator              = "UseDevelopmentEmulator"
	)
	var errs []error

	csp := ConnectionStringProperties{}

	splits := strings.Split(connStr, ";")

	for _, split := range splits {
		if split == "" {
			continue
		}

		keyAndValue := strings.SplitN(split, "=", 2)
		if len(keyAndValue) < 2 {
			return ConnectionStringProperties{}, errors.New("failed parsing connection string due to unmatched key value separated by '='")
		}

		// if a key value pair has `=` in the value, recombine them
		key := keyAndValue[0]
		value := strings.Join(keyAndValue[1:], "=")
		switch {
		case strings.EqualFold(endpointKey, key):
			u, err := url.Parse(value)
			if err != nil {
				return ConnectionStringProperties{}, errors.New("failed parsing connection string due to an incorrectly formatted Endpoint value")
			}
			csp.Endpoint = value
			csp.FullyQualifiedNamespace = u.Host
		case strings.EqualFold(sharedAccessKeyNameKey, key):
			csp.SharedAccessKeyName = &value
		case strings.EqualFold(sharedAccessKeyKey, key):
			csp.SharedAccessKey = &value
		case strings.EqualFold(entityPathKey, key):
			csp.EntityPath = &value
		case strings.EqualFold(sharedAccessSignatureKey, key):
			csp.SharedAccessSignature = &value
		case strings.EqualFold(useEmulator, key):
			v, err := strconv.ParseBool(value)

			if err != nil {
				return ConnectionStringProperties{}, err
			}

			csp.Emulator = v
		}
	}

	if csp.Emulator {
		endpointParts := strings.SplitN(csp.Endpoint, ":", 3) // allow for a port, if it exists.

		if len(endpointParts) < 2 || endpointParts[0] != "sb" {
			// there should always be at least two parts "sb:" and "//<emulator hostname>"
			// with an optional 3rd piece that's the port "1111".
			// (we don't need to validate it's a valid host since it's been through url.Parse() above)
			// return ConnectionStringProperties{}, fmt.Errorf("UseDevelopmentEmulator=true can only be used with sb://<emulator hostname> or sb://<emulator hostname>:<port number>, not %s", csp.Endpoint)
			errs = append(errs, fmt.Errorf("UseDevelopmentEmulator=true can only be used with sb://<emulator hostname> or sb://<emulator hostname>:<port number>, not %s", csp.Endpoint))
		}
	}

	if csp.FullyQualifiedNamespace == "" {
		errs = append(errs, fmt.Errorf("key %q must not be empty", endpointKey))
	}

	if (csp.SharedAccessSignature == nil || *csp.SharedAccessSignature == "") && (csp.SharedAccessKeyName == nil || *csp.SharedAccessKeyName == "") {
		// return ConnectionStringProperties{}, fmt.Errorf("key %q must not be empty", sharedAccessKeyNameKey)
		errs = append(errs, fmt.Errorf("key %q and %q must not be empty", sharedAccessKeyNameKey, sharedAccessSignatureKey))
	}

	if (csp.SharedAccessSignature == nil || *csp.SharedAccessSignature == "") && (csp.SharedAccessKey == nil || *csp.SharedAccessKey == "") {
		// return ConnectionStringProperties{}, fmt.Errorf("key %q or %q cannot both be empty", sharedAccessKeyKey, sharedAccessSignatureKey)
		errs = append(errs, fmt.Errorf("key %q or %q cannot both be empty", sharedAccessKeyKey, sharedAccessSignatureKey))
	}

	if len(errs) > 0 {
		return ConnectionStringProperties{}, errors.Join(errs...)
	}

	return csp, nil
}
