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
// and trailing slashes of the path are ignored.
func parseReference(content string) (path, field string, err error) {
	// "@" is reserved to select a named secret_stores connection.
	if strings.Contains(content, "@") {
		return "", "", errors.New(`named connections are not supported yet, percent-encode "@" as %40`)
	}
	rawPath, rawField, ok := strings.Cut(content, "#")
	if !ok {
		return "", "", errors.New("must have the form vault://<path>#<field>")
	}
	if strings.Contains(rawField, "#") {
		return "", "", errors.New(`field must not contain "#", percent-encode it as %23`)
	}
	path, err = url.PathUnescape(strings.Trim(rawPath, "/"))
	if err != nil {
		return "", "", fmt.Errorf("invalid path: %w", err)
	}
	field, err = url.PathUnescape(rawField)
	if err != nil {
		return "", "", fmt.Errorf("invalid field: %w", err)
	}
	if strings.Trim(path, "/") == "" || field == "" {
		return "", "", errors.New("empty path or field")
	}
	return path, field, nil
}
