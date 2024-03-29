// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Code generated by beats/dev-tools/cmd/asset/asset.go - DO NOT EDIT.

package system

import (
	"github.com/elastic/beats/v7/libbeat/asset"
)

func init() {
	if err := asset.SetFields("auditbeat", "system", asset.ModuleFieldsPri, AssetSystem); err != nil {
		panic(err)
	}
}

// AssetSystem returns asset data.
// This is the base64 encoded zlib format compressed contents of module/system.
func AssetSystem() string {
	return "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"
}
