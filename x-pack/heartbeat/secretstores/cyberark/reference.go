// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cyberark

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// accountRef identifies a CyberArk account in a Safe (and optional Folder).
type accountRef struct {
	safe   string
	folder string
	object string
}

func (r accountRef) String() string {
	if r.folder != "" {
		return r.safe + "/" + r.folder + "/" + r.object
	}
	return r.safe + "/" + r.object
}

// parseReference splits content, the text of a reference after "cyberark://",
// into an accountRef (<safe>/<object> or <safe>/<folder>/<object>) and a
// field, all percent-decoded. Leading and trailing slashes of the path are
// ignored.
func parseReference(content string) (ref accountRef, field string, err error) {
	// "@" is reserved to select a named secret_stores connection.
	if strings.Contains(content, "@") {
		return accountRef{}, "", errors.New(`named connections are not supported yet, percent-encode "@" as %40`)
	}
	rawPath, rawField, ok := strings.Cut(content, "#")
	if !ok {
		return accountRef{}, "", errors.New("must have the form cyberark://<safe>/<object>#<field>")
	}
	if strings.Contains(rawField, "#") {
		return accountRef{}, "", errors.New(`field must not contain "#", percent-encode it as %23`)
	}

	rawSegments := strings.Split(strings.Trim(rawPath, "/"), "/")
	if len(rawSegments) < 2 || len(rawSegments) > 3 {
		return accountRef{}, "", errors.New("must have the form cyberark://<safe>/<object>#<field> or cyberark://<safe>/<folder>/<object>#<field>")
	}
	segments := make([]string, len(rawSegments))
	for i, rawSeg := range rawSegments {
		seg, err := url.PathUnescape(rawSeg)
		if err != nil {
			return accountRef{}, "", fmt.Errorf("invalid path: %w", err)
		}
		seg = strings.Trim(seg, "/")
		if seg == "" {
			return accountRef{}, "", errors.New("empty path or field")
		}
		if seg == "." || seg == ".." || strings.Contains(seg, "/") {
			return accountRef{}, "", errors.New(`path must not have "." or ".." segments`)
		}
		segments[i] = seg
	}

	field, err = url.PathUnescape(rawField)
	if err != nil {
		return accountRef{}, "", fmt.Errorf("invalid field: %w", err)
	}
	if field == "" {
		return accountRef{}, "", errors.New("empty path or field")
	}

	if len(segments) == 2 {
		return accountRef{safe: segments[0], object: segments[1]}, field, nil
	}
	return accountRef{safe: segments[0], folder: segments[1], object: segments[2]}, field, nil
}
