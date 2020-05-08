// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build dev

package release

import "os"

func init() {
	envVersion, ok := os.LookupEnv("BEATS_VERSION")
	if ok {
		version = envVersion
	}
}
