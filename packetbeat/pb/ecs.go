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

package pb

import (
	"github.com/elastic/ecs/code/go/ecs"
)

// Fixes for non-array datatypes
// =============================
//
// Code at github.com/elastic/ecs/code/go/ecs has some fields as string
// when they should be []string.
//
// Once the code generator is fixed, this code will no longer compile
// which reminds us to strip out the overrides below
var (
	compileTimeUpgradeCheckEvent = ecs.Event{
		Type: "remove this when we upgrade ECS",
	}
	compileTimeUpgradeCheckRelated = ecs.Related{
		User: "remove this when we upgrade ECS",
	}
)

type ecsEvent struct {
	ecs.Event `ecs:",inline"`
	// overridden because this needs to be an array
	Category []string `ecs:"category"`
	// overridden because this needs to be an array
	Type []string `ecs:"type"`
}

type ecsRelated struct {
	ecs.Related `ecs:",inline"`
	// overridden because this needs to be an array
	IP []string `ecs:"ip"`
	// overridden because this needs to be an array
	User []string `ecs:"user"`
	// overridden because this needs to be an array
	Hash []string `ecs:"hash"`
	// overridden because this needs to be an array
	Hosts []string `ecs:"hosts"`

	// for de-dup
	ipSet   map[string]struct{}
	userSet map[string]struct{}
	hostSet map[string]struct{}
}
