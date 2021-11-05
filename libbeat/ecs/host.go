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

// A host is defined as a general computing instance.
// ECS host.* fields should be populated with details about the host on which
// the event happened, or from which the measurement was taken. Host types
// include hardware, virtual machines, Docker containers, and Kubernetes nodes.
type Host struct {
	// Hostname of the host.
	// It normally contains what the `hostname` command returns on the host
	// machine.
	Hostname string `ecs:"hostname"`

	// Name of the host.
	// It can contain what `hostname` returns on Unix systems, the fully
	// qualified domain name, or a name specified by the user. The sender
	// decides which value to use.
	Name string `ecs:"name"`

	// Unique host id.
	// As hostname is not always unique, use values that are meaningful in your
	// environment.
	// Example: The current usage of `beat.name`.
	ID string `ecs:"id"`

	// Host ip addresses.
	IP string `ecs:"ip"`

	// Host MAC addresses.
	// The notation format from RFC 7042 is suggested: Each octet (that is,
	// 8-bit byte) is represented by two [uppercase] hexadecimal digits giving
	// the value of the octet as an unsigned integer. Successive octets are
	// separated by a hyphen.
	MAC string `ecs:"mac"`

	// Type of host.
	// For Cloud providers this can be the machine type like `t2.medium`. If
	// vm, this could be the container, for example, or other information
	// meaningful in your environment.
	Type string `ecs:"type"`

	// Seconds the host has been up.
	Uptime int64 `ecs:"uptime"`

	// Operating system architecture.
	Architecture string `ecs:"architecture"`

	// Name of the domain of which the host is a member.
	// For example, on Windows this could be the host's Active Directory domain
	// or NetBIOS domain name. For Linux this could be the domain of the host's
	// LDAP provider.
	Domain string `ecs:"domain"`

	// Percent CPU used which is normalized by the number of CPU cores and it
	// ranges from 0 to 1.
	// Scaling factor: 1000.
	// For example: For a two core host, this value should be the average of
	// the two cores, between 0 and 1.
	CpuUsage float64 `ecs:"cpu.usage"`

	// The total number of bytes (gauge) read successfully (aggregated from all
	// disks) since the last metric collection.
	DiskReadBytes int64 `ecs:"disk.read.bytes"`

	// The total number of bytes (gauge) written successfully (aggregated from
	// all disks) since the last metric collection.
	DiskWriteBytes int64 `ecs:"disk.write.bytes"`

	// The number of bytes received (gauge) on all network interfaces by the
	// host since the last metric collection.
	NetworkIngressBytes int64 `ecs:"network.ingress.bytes"`

	// The number of packets (gauge) received on all network interfaces by the
	// host since the last metric collection.
	NetworkIngressPackets int64 `ecs:"network.ingress.packets"`

	// The number of bytes (gauge) sent out on all network interfaces by the
	// host since the last metric collection.
	NetworkEgressBytes int64 `ecs:"network.egress.bytes"`

	// The number of packets (gauge) sent out on all network interfaces by the
	// host since the last metric collection.
	NetworkEgressPackets int64 `ecs:"network.egress.packets"`
}
