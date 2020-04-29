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

package gotype

import (
	"reflect"
)

type initUnfoldOptions struct {
	unfoldFns map[reflect.Type]reflUnfolder
}

type UnfoldOption func(*initUnfoldOptions) error

func applyUnfoldOpts(opts []UnfoldOption) (i initUnfoldOptions, err error) {
	for _, o := range opts {
		if err = o(&i); err != nil {
			break
		}
	}
	return i, err
}

// Unfolders accepts a list of primitive, processing, or stateful unfolders.
//
// Primitive unfolder must implement a function matching the type: func(to *Target, from P) error
// Where to is an arbitrary go type that the result should be written to and
// P must be one of: bool, string, uint(8|16|32|64), int(8|16|32|64), float(32|64)
//
// Processing unfolders first unfold a structure into a temporary structure, followed
// by a post-processing function used to fill in the original target. Processing unfolders
// for type T have the signature:
// func(to *T) (cell interface{}, process func(to *T, cell interface{}) error)
//
// A processing unfolder returns a temporary value for unfolding. The Unfolder will process
// the temporary value (held in cell), like any regular supported value.
// The process function is executed if the parsing step did succeed.
// The address to the target structure and the original cell are reported to the process function,
// reducing the need for allocation storage on the heap in most simple cases.
//
// Stateful unfolders have the function signature: func(to *T) UnfoldState.
// The state returned by the initialization function is used for parsing.
// Although stateful unfolders allow for the most complex unfolding possible,
// they add the most overhead in managing state and allocations. If possible
// prefer primitive unfolders, followed by processing unfolder.
func Unfolders(in ...interface{}) UnfoldOption {
	unfolders, err := makeUserUnfolderFns(in)
	if err != nil || len(unfolders) == 0 {
		return func(_ *initUnfoldOptions) error { return err }
	}

	return func(o *initUnfoldOptions) error {
		if o.unfoldFns == nil {
			o.unfoldFns = map[reflect.Type]reflUnfolder{}
		}

		for k, v := range unfolders {
			o.unfoldFns[k] = v
		}
		return nil
	}
}

func makeUserUnfolderFns(in []interface{}) (map[reflect.Type]reflUnfolder, error) {
	M := map[reflect.Type]reflUnfolder{}

	for _, cur := range in {
		if cur == nil {
			continue
		}

		t, unfolder, err := makeUserUnfolder(reflect.ValueOf(cur))
		if err != nil {
			return nil, err
		}

		M[t] = unfolder
	}

	return M, nil
}
