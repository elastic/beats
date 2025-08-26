// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build requirefips

package netflow

var reverseFlowsTestKeys = []string{"flow.id", "flow.locality"}

// stripCommunityID removes the community_id field from all flows when in fips mode
func stripCommunityID(tr *TestResult) {
	for _, flow := range tr.Flows {
		_ = flow.Delete("network.community_id")
	}
}
