// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package process

import "github.com/elastic/beats/v7/auditbeat/helper/hasher"

var defaultHashes = []hasher.HashType{hasher.SHA1}
