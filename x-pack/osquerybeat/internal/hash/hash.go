// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
)

// Calculate hash with optional writer to compbine, useful when streaming data to the disk
func Calculate(r io.Reader, w io.Writer) (string, error) {
	h := sha256.New()

	if w != nil {
		w = io.MultiWriter(h, w)
	} else {
		w = h
	}

	if _, err := io.Copy(w, r); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
