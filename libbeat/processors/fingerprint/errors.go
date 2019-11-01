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

package fingerprint

import (
	"errors"
	"fmt"
)

var errNoFields = errors.New("must specify at least one field")

type (
	errUnknownEncoding    struct{ encoding string }
	errUnknownMethod      struct{ method string }
	errConfigUnpack       struct{ cause error }
	errComputeFingerprint struct{ cause error }
	errMissingField       struct {
		field string
		cause error
	}
	errNonScalarField struct{ field string }
)

func makeErrUnknownEncoding(encoding string) errUnknownEncoding {
	return errUnknownEncoding{encoding}
}
func (e errUnknownEncoding) Error() string {
	return fmt.Sprintf("invalid encoding [%s]", e.encoding)
}

func makeErrUnknownMethod(method string) errUnknownMethod {
	return errUnknownMethod{method}
}
func (e errUnknownMethod) Error() string {
	return fmt.Sprintf("invalid fingerprinting method [%s]", e.method)
}

func makeErrConfigUnpack(cause error) errConfigUnpack {
	return errConfigUnpack{cause}
}
func (e errConfigUnpack) Error() string {
	return fmt.Sprintf("failed to unpack %v processor configuration: %v", processorName, e.cause)
}

func makeErrComputeFingerprint(cause error) errComputeFingerprint {
	return errComputeFingerprint{cause}
}
func (e errComputeFingerprint) Error() string {
	return fmt.Sprintf("failed to compute fingerprint: %v", e.cause)
}

func makeErrMissingField(field string, cause error) errMissingField {
	return errMissingField{field, cause}
}
func (e errMissingField) Error() string {
	return fmt.Sprintf("failed to find field [%v] in event: %v", e.field, e.cause)
}

func makeErrNonScalarField(field string) errNonScalarField {
	return errNonScalarField{field}
}
func (e errNonScalarField) Error() string {
	return fmt.Sprintf("cannot compute fingerprint using non-scalar field [%v]", e.field)
}
