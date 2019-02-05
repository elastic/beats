// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package system

import (
	"crypto/sha256"
	"encoding/hex"
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

// Sum returns the hash as a string.
func (h *EntityHash) Sum() string {
	return hex.EncodeToString(h.Hash.Sum(nil))
}
