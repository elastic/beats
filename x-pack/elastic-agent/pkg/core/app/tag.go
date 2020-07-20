// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package app

// Tag is a tag for specifying metadata related
// to a process.
type Tag string

// TagSidecar tags a sidecar process
const TagSidecar = "sidecar"

// Taggable is an object containing tags.
type Taggable interface {
	Tags() map[Tag]string
}

// IsSidecar returns true if tags contains sidecar flag.
func IsSidecar(descriptor Taggable) bool {
	tags := descriptor.Tags()
	_, isSidecar := tags[TagSidecar]
	return isSidecar
}
