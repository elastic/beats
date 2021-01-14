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

package ratelimit

import (
	"fmt"
	"strconv"
	"strings"
)

type unit string

const (
	unitPerSecond unit = "s"
	unitPerMinute unit = "m"
	unitPerHour   unit = "h"
)

type rate struct {
	value float64
	unit  unit
}

// Unpack creates a rate from the given string
func (l *rate) Unpack(str string) error {
	parts := strings.Split(str, "/")
	if len(parts) != 2 {
		return fmt.Errorf(`rate in invalid format: %v. Must be specified as "number/unit"`, str)
	}

	valueStr := strings.TrimSpace(parts[0])
	unitStr := strings.TrimSpace(parts[1])

	v, err := strconv.ParseFloat(valueStr, 8)
	if err != nil {
		return fmt.Errorf(`rate's value component is not numeric: %v`, valueStr)
	}

	if allowed := []unit{unitPerSecond, unitPerMinute, unitPerHour}; !contains(allowed, unitStr) {
		allowedStrs := make([]string, len(allowed))
		for _, a := range allowed {
			allowedStrs = append(allowedStrs, "/"+string(a))
		}

		return fmt.Errorf(`rate's unit component must be specified as one of: %v`, strings.Join(allowedStrs, ","))
	}

	u := unit(unitStr)

	l.value = v
	l.unit = u

	return nil
}

func (l *rate) valuePerSecond() float64 {
	switch l.unit {
	case unitPerSecond:
		return l.value
	case unitPerMinute:
		return l.value / 60
	case unitPerHour:
		return l.value / (60 * 60)
	}

	return 0
}

func contains(allowed []unit, candidate string) bool {
	for _, a := range allowed {
		if candidate == string(a) {
			return true
		}
	}

	return false
}
