// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package system

import (
	"crypto/sha256"
	"encoding/base64"
	"hash"
)

// EntityHash calculates a standard entity hash.
type EntityHash struct {
	hash.Hash
}

// NewEntityHash creates a new EntityHash.
func NewEntityHash() EntityHash {
	return EntityHash{sha256.New()}
}

// Sum returns the base64 representation of the hash,
// truncated to 12 bytes.
func (h *EntityHash) Sum() string {
	hash := h.Hash.Sum(nil)
	if len(hash) > 12 {
		hash = hash[:12]
	}
	return base64.RawStdEncoding.EncodeToString(hash)
}
