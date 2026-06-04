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

package memlog

// Config defines the user-facing configuration for the memlog storage backend.
type Config struct {
	// CheckpointSize is the registry file size threshold (in bytes) that
	// triggers a checkpoint. Larger values reduce checkpoint frequency at the
	// cost of a larger registry file. Minimum: 10 MB. Default: 10 MB.
	CheckpointSize uint64 `config:"checkpoint_size" validate:"min=10485760"`
}

// DefaultConfig returns the default memlog configuration.
func DefaultConfig() Config {
	return Config{CheckpointSize: defaultCheckpointSize}
}
