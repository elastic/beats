// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

// SafeStrp makes handling AWS *string types easier.
// The AWS lib never returns plain strings, always using pointers, probably for memory efficiency reasons.
// This is a bit odd, because strings are just pointers into byte arrays, however this is the choice they've made.
// This will return the plain version of the given string or an empty string if the pointer is null
func SafeStrp(strp *string) string {
	if strp == nil {
		return ""
	}

	return *strp
}
