// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package download

// Downloader is an interface allowing download of an artifact
type Downloader interface {
	Download(programName, version string) (string, error)
}
