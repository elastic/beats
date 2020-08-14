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

package cron

import (
	"time"

	"github.com/gorhill/cronexpr"
)

type Schedule cronexpr.Expression

func MustParse(in string) *Schedule {
	s, err := Parse(in)
	if err != nil {
		panic(err)
	}
	return s
}

func Parse(in string) (*Schedule, error) {
	expr, err := cronexpr.Parse(in)
	return (*Schedule)(expr), err
}

func (s *Schedule) Next(t time.Time) time.Time {
	expr := (*cronexpr.Expression)(s)
	return expr.Next(t)
}

func (s *Schedule) Unpack(str string) error {
	tmp, err := Parse(str)
	if err == nil {
		*s = *tmp
	}
	return err
}

// RunOnInit returns false for interval schedulers.
func (s *Schedule) RunOnInit() bool {
	return false
}
