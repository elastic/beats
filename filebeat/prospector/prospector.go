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

// Package prospector allows to define new way of reading data in Filebeat
// Deprecated: See the input package
package prospector

import "github.com/elastic/beats/filebeat/input"

// Prospectorer defines how to read new data
// Deprecated: See input.input
type Prospectorer = input.Input

// Runner encapsulate the lifecycle of a prospectorer
// Deprecated: See input.Runner
type Runner = input.Runner

// Context wrapper for backward compatibility
// Deprecated: See input.Context
type Context = input.Context

// Factory wrapper for backward compatibility
// Deprecated: See input.Factory
type Factory = input.Factory

// Register wrapper for backward compatibility
// Deprecated: See input.Register
var Register = input.Register

// GetFactory wrapper for backward compatibility
// Deprecated: See input.GetFactory
var GetFactory = input.GetFactory

// New wrapper for backward compatibility
// Deprecated: see input.New
var New = input.New

// NewRunnerFactory wrapper for backward compatibility
// Deprecated: see input.NewRunnerFactory
var NewRunnerFactory = input.NewRunnerFactory
