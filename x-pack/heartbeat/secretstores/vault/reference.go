// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package vault

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// parseReference splits content, the text of a reference after "vault://",
// into a KV path and a field: <path>#<field>, both percent-decoded. Leading
// and trailing slashes of the path are ignored. The path can't have "." or
// ".." segments: the Vault client cleans the request path, so they would
// read outside the KV mount.
func parseReference(content string) (path, field string, err error) {
	// The connection name is removed before, a second "@" is not valid.
	if strings.Contains(content, "@") {
		return "", "", errors.New(`must not contain "@", percent-encode it as %40`)
	}
	rawPath, rawField, ok := strings.Cut(content, "#")
	if !ok {
		return "", "", errors.New("must have the form vault://<name>@<path>#<field>")
	}
	if strings.Contains(rawField, "#") {
		return "", "", errors.New(`field must not contain "#", percent-encode it as %23`)
	}
	path, err = url.PathUnescape(rawPath)
	if err != nil {
		return "", "", fmt.Errorf("invalid path: %w", err)
	}
	path = strings.Trim(path, "/")
	for segment := range strings.SplitSeq(path, "/") {
		if segment == "." || segment == ".." {
			return "", "", errors.New(`path must not have "." or ".." segments`)
		}
	}
	field, err = url.PathUnescape(rawField)
	if err != nil {
		return "", "", fmt.Errorf("invalid field: %w", err)
	}
	if path == "" || field == "" {
		return "", "", errors.New("empty path or field")
	}
	return path, field, nil
}
