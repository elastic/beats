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

package memlog

import "github.com/elastic/beats/libbeat/common"

type (
	op interface {
		name() string
	}

	opInsertWith struct {
		K string
		V common.MapStr
	}

	opUpdate struct {
		K string
		V common.MapStr
	}

	opRemove struct {
		K string
	}

	opBegin struct {
		ID uint64
	}

	opCommit struct {
		ID uint64
	}

	opRollback struct {
		ID uint64
	}
)

// operation type names
const (
	opValInsert   = "insert"
	opValUpdate   = "update"
	opValRemove   = "remove"
	opValBegin    = "begin"
	opValCommit   = "commit"
	opValRollback = "rollback"
)

func (*opInsertWith) name() string { return opValInsert }

func (*opUpdate) name() string { return opValUpdate }

func (*opRemove) name() string { return opValRemove }

func (*opBegin) name() string { return opValBegin }

func (*opCommit) name() string { return opValCommit }

func (*opRollback) name() string { return opValRollback }

func mergeKVOp(prev, curr op) (first, second op) {
	switch o := curr.(type) {
	case *opInsertWith, *opRemove:
		return curr, nil // insert/remove overwrites
	case *opUpdate:
		return mergeUpdateOp(prev, o)
	default:
		return prev, curr
	}
}

func mergeUpdateOp(prev op, curr *opUpdate) (first, second op) {
	switch o := prev.(type) {
	case *opInsertWith:
		o.V.DeepUpdate(curr.V)
		return o, nil
	case *opUpdate:
		o.V.DeepUpdate(curr.V)
		return o, nil
	case *opRemove:
		return curr, nil
	default:
		return prev, curr
	}
}
