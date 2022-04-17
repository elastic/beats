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

// Package module contains the low-level utilities for running Metricbeat
// modules and metricsets. This is useful for building your own tool that
// has a module and sub-module concept. If you want to reuse the whole
// Metricbeat framework see the github.com/menderesk/beats/v7/metricbeat/beater
// package that provides a higher level interface.
//
// This contains the tools for instantiating modules, running them, and
// connecting their outputs to the Beat's output pipeline.
package module
