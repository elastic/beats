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

package v2

import conf "github.com/elastic/elastic-agent-libs/config"

// Redirector is an optional interface that an InputManager can implement
// to redirect input creation to a different input type. The Loader checks
// for this interface before calling Create.
//
// When an InputManager implements Redirector, the Loader calls Redirect
// first. If Redirect returns a non-empty target type, the Loader resolves
// the target plugin from its registry and calls its Create with the
// translated config. Only one redirect hop is allowed; the target's
// Redirector (if any) is not consulted.
//
// If Redirect returns an empty target type, the Loader proceeds to call
// Create on the original InputManager as usual.
type Redirector interface {
	// Redirect examines the config and returns the name of the target
	// input type and a translated config if this input should be run as
	// another type. Return ("", nil, nil) for no redirect.
	Redirect(cfg *conf.C) (targetType string, translatedCfg *conf.C, err error)
}
