// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package download

import "context"

// Downloader is an interface allowing download of an artifact
type Downloader interface {
	Download(ctx context.Context, programName, version string) (string, error)
}
