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

// Package ucfg provides a common representation for hierarchical configurations.
//
// The common representation provided by the Config type can be used with different
// configuration file formats like XML, JSON, HSJSON, YAML, or TOML.
//
// Config provides a low level and a high level interface for reading settings
// with additional features like custom unpackers, validation and capturing
// sub-configurations for deferred interpretation, lazy intra-configuration
// variable expansion, and OS environment variable expansion.
package ucfg
