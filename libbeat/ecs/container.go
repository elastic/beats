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

	// Container cpu usage
	// Percent CPU used which is normalized by the number of CPU cores
	// and it ranges from 0 to 1. Scaling factor: 1000.
	// Scaling factor: 1000.
	CPUUsage float64 `ecs:"cpu.usage"`

	// The total number of bytes (gauge) read successfully (aggregated
	// from all disks) since the last metric collection.
	DiskReadBytes int64 `ecs:"disk.read.bytes"`

	// The total number of bytes (gauge) written successfully (aggregated from
	// all disks) since the last metric collection.
	DiskWriteBytes int64 `ecs:"disk.write.bytes"`

	// Memory usage percentage and it ranges from 0 to 1. Scaling factor: 1000.
	MemoryUsage float64 `ecs:"memory.usage"`

	// The number of bytes received (gauge) on all network interfaces by the
	// container since the last metric collection.
	NetworkIngressBytes int64 `ecs:"network.ingress.bytes"`

	// The number of bytes (gauge) sent out on all network interfaces by the
	// container since the last metric collection.
	NetworkEgressBytes int64 `ecs:"network.egress.bytes"`

	// Image labels.
	Labels map[string]interface{} `ecs:"labels"`
}
