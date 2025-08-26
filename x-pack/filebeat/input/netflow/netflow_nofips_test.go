// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package netflow

var reverseFlowsTestKeys = []string{"flow.id", "flow.locality", "network.community_id"}

// stripCommunityID is a nop when requirefips is not passed.
func stripCommunityID(tr *TestResult) {}
