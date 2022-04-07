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

package host

import (
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/monitoring"
	"github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"
)

// MapHostInfo converts the HostInfo to a MapStr based on ECS.
func MapHostInfo(info types.HostInfo) common.MapStr {
	data := common.MapStr{
		"host": common.MapStr{
			"hostname":     info.Hostname,
			"architecture": info.Architecture,
			"os": common.MapStr{
				"platform": info.OS.Platform,
				"version":  info.OS.Version,
				"family":   info.OS.Family,
				"name":     info.OS.Name,
				"kernel":   info.KernelVersion,
			},
		},
	}

	// Optional params
	if info.UniqueID != "" {
		data.Put("host.id", info.UniqueID)
	}
	if info.Containerized != nil {
		data.Put("host.containerized", *info.Containerized)
	}
	if info.OS.Codename != "" {
		data.Put("host.os.codename", info.OS.Codename)
	}
	if info.OS.Build != "" {
		data.Put("host.os.build", info.OS.Build)
	}
	if info.OS.Type != "" {
		data.Put("host.os.type", info.OS.Type)
	}
	return data
}

// ReportInfo reports the HostInfo to monitoring.
func ReportInfo(_ monitoring.Mode, V monitoring.Visitor) {
	V.OnRegistryStart()
	defer V.OnRegistryFinished()

	h, err := sysinfo.Host()
	if err != nil {
		return
	}
	info := h.Info()

	monitoring.ReportString(V, "hostname", info.Hostname)
	monitoring.ReportString(V, "architecture", info.Architecture)
	monitoring.ReportNamespace(V, "os", func() {
		monitoring.ReportString(V, "platform", info.OS.Platform)
		monitoring.ReportString(V, "version", info.OS.Version)
		monitoring.ReportString(V, "family", info.OS.Family)
		monitoring.ReportString(V, "name", info.OS.Name)
		monitoring.ReportString(V, "kernel", info.KernelVersion)

		if info.OS.Codename != "" {
			monitoring.ReportString(V, "codename", info.OS.Codename)
		}
		if info.OS.Build != "" {
			monitoring.ReportString(V, "build", info.OS.Build)
		}
	})

	if info.UniqueID != "" {
		monitoring.ReportString(V, "id", info.UniqueID)
	}
	if info.Containerized != nil {
		monitoring.ReportBool(V, "containerized", *info.Containerized)
	}
}
