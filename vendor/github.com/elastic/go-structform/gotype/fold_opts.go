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

import "reflect"

type initFoldOptions struct {
	foldFns map[reflect.Type]reFoldFn
}

type FoldOption func(*initFoldOptions) error

func applyFoldOpts(opts []FoldOption) (i initFoldOptions, err error) {
	for _, o := range opts {
		if err = o(&i); err != nil {
			break
		}
	}
	return i, err
}

func Folders(in ...interface{}) FoldOption {
	folders, err := makeUserFoldFns(in)
	if err != nil {
		return func(_ *initFoldOptions) error { return err }
	}

	if len(folders) == 0 {
		return func(*initFoldOptions) error { return nil }
	}

	return func(o *initFoldOptions) error {
		if o.foldFns == nil {
			o.foldFns = map[reflect.Type]reFoldFn{}
		}

		for k, v := range folders {
			o.foldFns[k] = v
		}
		return nil
	}
}

func makeUserFoldFns(in []interface{}) (map[reflect.Type]reFoldFn, error) {
	M := map[reflect.Type]reFoldFn{}

	for _, v := range in {
		fn := reflect.ValueOf(v)
		fptr, err := makeUserFoldFn(fn)
		if err != nil {
			return nil, err
		}

		ta0 := fn.Type().In(0)
		M[ta0] = liftUserPtrFn(fptr)
		M[ta0.Elem()] = liftUserValueFn(fptr)
	}

	return M, nil
}
