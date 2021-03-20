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

package reason

type Reason interface {
	error
	Type() string
	Code() string
	Unwrap() error
}

func NewResolve(err error) Reason {
	return NewCustReason(err, "io", "could_not_resolve_host")
}

func NewCustReason(err error, typ string, code string) CustReason {
	return CustReason{Err: err, Message: err.Error(), Typ: typ, CodeV: code}
}

type CustReason struct {
	Err error
	Message string `json:"message"`
	CodeV string `json:"code"`
	Typ string `json:"type"`
}

func (c CustReason) Error() string {
	return c.Err.Error()
}

func (c CustReason) Type() string {
	return c.Typ
}

func (c CustReason) Code() string {
	return c.CodeV
}

func (c CustReason) Unwrap() error {
	return c.Err
}
