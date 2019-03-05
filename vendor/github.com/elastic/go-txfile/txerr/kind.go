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

package txerr

// Is checks if any error in the error tree matches `kind`.
func Is(kind error, in error) bool {
	return FindKind(in, kind) != nil
}

// GetKind returns the first error kind found in the error tree.
func GetKind(in error) error {
	err := FindKindIf(in, func(_ error) bool { return true })
	if err == nil {
		return nil
	}
	return err.(withKind).Kind()
}

// FindKind returns the first error that matched `kind`.
func FindKind(in error, kind error) error {
	return FindKindIf(in, func(k error) bool { return k == kind })
}

// FindKindIf returns the first error with a kind that fulfills the user predicate
func FindKindIf(in error, fn func(kind error) bool) error {
	return FindErrWith(in, func(in error) bool {
		if err, ok := in.(withKind); ok {
			if k := err.Kind(); k != nil {
				return fn(err.Kind())
			}
		}
		return false
	})
}

func directKind(in error) error {
	if err, ok := in.(withKind); ok {
		return err.Kind()
	}
	return nil
}
