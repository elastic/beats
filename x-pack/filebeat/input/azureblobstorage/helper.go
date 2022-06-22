// // Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// // or more contributor license agreements. Licensed under the Elastic License;
// // you may not use this file except in compliance with the Elastic License.

// //go:build !aix
// // +build !aix

package azureblobstorage

import (
	"fmt"
	"strings"
)

func fetchJobID(jobCounter int, containerName string, blobName string) string {
	jobID := fmt.Sprintf("%s-%s-%d", strings.ToUpper(containerName), strings.ToUpper(blobName), jobCounter)

	return jobID
}
