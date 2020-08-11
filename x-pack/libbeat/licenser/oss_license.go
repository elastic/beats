// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package licenser

import "github.com/gofrs/uuid"

func mustUUIDV4() uuid.UUID {
	uuid, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	return uuid
}

// OSSLicense default license to use.
var (
	OSSLicense = &License{
		UUID:   mustUUIDV4().String(),
		Type:   OSS,
		Status: Active,
	}
)
