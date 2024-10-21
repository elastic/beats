// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package common

import "github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/define"

// OSBatch defines the mapping between a SupportedOS and a define.Batch.
type OSBatch struct {
	// ID is the unique ID for the batch.
	ID string
	// LayoutOS provides all the OS information to create an instance.
	OS SupportedOS
	// Batch defines the batch of tests to run on this layout.
	Batch define.Batch
	// Skip defines if this batch will be skipped because no supported layout exists yet.
	Skip bool
}
