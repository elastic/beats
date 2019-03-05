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

// IsOp checks if any error in the error tree is caused by `op`.
func IsOp(op string, in error) bool {
	return FindOp(in, op) != nil
}

// GetOp returns the first errors it's Op value.
func GetOp(in error) string {
	err := FindErrWith(in, func(in error) bool {
		if err, ok := in.(withOp); ok {
			return err.Op() != ""
		}
		return false
	})

	if err == nil {
		return ""
	}
	return err.(withOp).Op()
}

// FindOp returns the first error with the given `op` value.
func FindOp(in error, op string) error {
	return FindErrWith(in, func(in error) bool {
		if err, ok := in.(withOp); ok {
			return err.Op() == op
		}
		return false
	})
}

func directOp(in error) string {
	if err, ok := in.(withOp); ok {
		return err.Op()
	}
	return ""
}
