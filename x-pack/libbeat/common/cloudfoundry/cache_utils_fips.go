// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build requirefips

package cloudfoundry

import (
	"encoding/base64"
	"strconv"

	"github.com/cespare/xxhash/v2"
)

// sanitizeCacheName returns a unique string that can be used safely as part of a file name
func sanitizeCacheName(name string) string {
	h := xxhash.Sum64([]byte(name))
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatUint(h, 10)))
}
