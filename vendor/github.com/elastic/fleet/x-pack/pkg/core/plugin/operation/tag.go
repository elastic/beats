// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

// Tag is a tag for specifying metadata related
// to a process.
type Tag string

// TagSidecarOf tags a sidecar process and identifies
// a process which is sidecard-ed.
// Example:
// - p1: filebeat 	- watches apache logs
// - p2: metricbeat - waches p1
// p1.Tags = []
// p2.Tags = [{ "sidecar-of": "filebeat" }]
const TagSidecarOf = "sidecar-of"
