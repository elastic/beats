// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build requirefips

package cloudfoundry

import (
	"crypto/sha256"
	"encoding/base64"
)

// sanitizeCacheName returns a unique string that can be used safely as part of a file name
func sanitizeCacheName(name string) string {
	hash := sha256.Sum224([]byte(name))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
