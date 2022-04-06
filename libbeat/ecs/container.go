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

package ecs

// Container fields are used for meta information about the specific container
// that is the source of information.
// These fields help correlate data based containers from any runtime.
type Container struct {
	// Runtime managing this container.
	Runtime string `ecs:"runtime"`

	// Unique container id.
	ID string `ecs:"id"`

	// Name of the image the container was built on.
	ImageName string `ecs:"image.name"`

	// Container image tags.
	ImageTag string `ecs:"image.tag"`

	// Container name.
	Name string `ecs:"name"`

	// Image labels.
	Labels map[string]interface{} `ecs:"labels"`
}
