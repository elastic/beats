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

package eventlog

// retry invokes the retriable function. If the retriable function returns an
// error then the corrective action function is invoked and passed the error.
// The correctiveAction function should attempt to correct the error so that
// retriable can be invoked again.
func retry(retriable func() error, correctiveAction func(error) error) error {
	err := retriable()
	if err != nil {
		caErr := correctiveAction(err)
		if caErr != nil {
			// Something went wrong, return original error.
			return err
		}

		retryErr := retriable()
		if retryErr != nil {
			// The second attempt failed, return original error.
			return err
		}
	}

	return nil
}
