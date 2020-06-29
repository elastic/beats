// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package release

import (
	"net/url"
	"strconv"
	"sync"
	"time"

	libbeatVersion "github.com/elastic/beats/v7/libbeat/version"
)

// snapshot is a flag marking build as a snapshot.
var snapshot = ""

// escPgp is escaped content of pgp bytes
var escPgp string

// pgp bytes is a packed in public gpg key
var pgpBytes []byte

// allowEmptyPgp is used as a debug flag and allows working
// without valid pgp
var allowEmptyPgp string

// Commit returns the current build hash or unknown if it was not injected in the build process.
func Commit() string {
	return libbeatVersion.Commit()
}

// BuildTime returns the build time of the binaries.
func BuildTime() time.Time {
	return libbeatVersion.BuildTime()
}

// Version returns the version of the application.
func Version() string {
	return libbeatVersion.GetDefaultVersion()
}

// Snapshot returns true if binary was built as snapshot.
func Snapshot() bool {
	val, err := strconv.ParseBool(snapshot)
	return err == nil && val
}

// PGP return pgpbytes and a flag describing whether or not no pgp is valid.
func PGP() (bool, []byte) {
	var pgpLoader sync.Once
	isEmptyAllowed := allowEmptyPgp == "true"

	pgpLoader.Do(func() {
		// initial sanity build check
		if len(escPgp) == 0 && !isEmptyAllowed {
			panic("GPG key is not present but required")
		}

		if len(escPgp) > 0 {
			if unescaped, err := url.PathUnescape(escPgp); err == nil {
				pgpBytes = []byte(unescaped)
			}
		}
	})

	return isEmptyAllowed, pgpBytes
}
