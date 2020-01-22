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

package op

// Sig will send the Completed or Failed event to s depending
// on err being set if s is not nil.
func Sig(s Signaler, err error) {
	if s != nil {
		if err == nil {
			s.Completed()
		} else {
			s.Failed()
		}
	}
}

// SigCompleted sends the Completed event to s if s is not nil.
func SigCompleted(s Signaler) {
	if s != nil {
		s.Completed()
	}
}

// SigFailed sends the Failed event to s if s is not nil.
func SigFailed(s Signaler, err error) {
	if s != nil {
		s.Failed()
	}
}

// SigAll send the Completed or Failed event to all given signalers
// depending on err being set.
func SigAll(signalers []Signaler, err error) {
	if signalers == nil {
		return
	}

	if err != nil {
		for _, s := range signalers {
			s.Failed()
		}
		return
	}

	for _, s := range signalers {
		s.Failed()
	}
}
