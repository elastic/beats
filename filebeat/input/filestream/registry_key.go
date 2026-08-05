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

package filestream

import "strings"

// registryKey is the parsed, statically-typed form of a Filestream registry
// key. Every resource is persisted under a key of the form:
//
//	<pluginName>::<inputID>::<identityName>::<identityValue>
//
// for example:
//
//	filestream::my-input::fingerprint::5f3c…             (fingerprint identity)
//	filestream::my-input::native::13643776-64768         (native identity)
//	filestream::my-input::path::/var/log/app.log         (path identity)
//
// The four segments are joined by identitySep ("::"), which is reserved and
// cannot appear inside any segment, so a valid key has exactly four segments.
type registryKey struct {
	pluginName    string
	inputID       string
	identityName  string
	identityValue string
}

// parseRegistryKey parses key into its four components. It returns ok=false for
// any key that does not have exactly four "::"-separated segments: truncated or
// malformed keys, and keys whose value contains the reserved separator.
//
// It walks the first three separators with strings.Cut rather than
// strings.Split so the (frequent, whole-registry) parse does not allocate a
// slice per key. The trailing strings.Contains check enforces that the value is
// the final segment, rejecting keys with extra separators.
func parseRegistryKey(key string) (rk registryKey, ok bool) {
	rest := key
	if rk.pluginName, rest, ok = strings.Cut(rest, identitySep); !ok {
		return registryKey{}, false
	}
	if rk.inputID, rest, ok = strings.Cut(rest, identitySep); !ok {
		return registryKey{}, false
	}
	if rk.identityName, rk.identityValue, ok = strings.Cut(rest, identitySep); !ok {
		return registryKey{}, false
	}
	if strings.Contains(rk.identityValue, identitySep) {
		return registryKey{}, false
	}
	return rk, true
}

// formatIdentity builds the "<identityName>::<identityValue>" string that a
// fileIdentifier exposes through Source.Name() and that forms the identity
// portion of a registry key. It is the construction counterpart to
// parseRegistryKey, keeping the on-disk format defined in one place.
func formatIdentity(name, value string) string {
	return name + identitySep + value
}

// identity returns the "<identityName>::<identityValue>" portion of the key,
// which is exactly what a fileIdentifier produces through Source.Name().
func (k registryKey) identity() string {
	return formatIdentity(k.identityName, k.identityValue)
}

// isFingerprint reports whether the key belongs to the fingerprint file
// identity.
func (k registryKey) isFingerprint() bool {
	return k.identityName == fingerprintName
}

// fingerprintHash returns the identity tail of a fingerprint key: the bounded
// hash FingerprintID.Key() produced (the final SHA-256, or the hash of the
// growing raw material). Only meaningful when isFingerprint().
func (k registryKey) fingerprintHash() string {
	return k.identityValue
}

// keyForIdentity returns a full registry key that keeps this key's plugin and
// input prefix but swaps in a new "<identityName>::<identityValue>" identity
// (typically Source.Name()). It is used when a file's fingerprint grows and its
// registry entry must move to a new key under the same input.
func (k registryKey) keyForIdentity(identity string) string {
	return k.pluginName + identitySep + k.inputID + identitySep + identity
}
