// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package info

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
	"github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"
)

// List of variables available to be used in constraint definitions.
const (
	// `agent.id` is a generated (in standalone) or assigned (in fleet) agent identifier.
	agentIDKey = "agent.id"
	// `agent.version` specifies current version of an agent.
	agentVersionKey = "agent.version"

	// `os.family` defines a family of underlying operating system (e.g. redhat, debian, freebsd, windows).
	osFamilyKey = "os.family"
	// `os.kernel` specifies current version of a kernel in a semver format.
	osKernelKey = "os.kernel"
	// `os.platform` specifies platform agent is running on (e.g. centos, ubuntu, windows).
	osPlatformKey = "os.platform"
	// `os.version` specifies version of underlying operating system (e.g. 10.12.6).
	osVersionKey = "os.version"
	// `os.name` is a operating system name.
	// Currently we just normalize the name (i.e. macOS, Windows, Linux). See https://www.elastic.co/guide/en/ecs/current/ecs-os.html
	osNameKey = "os.name"
	// `os.full` is an operating system name, including the version or code name.
	osFullKey = "os.full"

	// `host.architecture` defines architecture of a host (e.g. x86_64, arm, ppc, mips).
	hostArchKey = "host.architecture"
	// `host.hostname` specifies hostname of the host.
	hostHostnameKey = "host.hostname"
	// `host.name` specifies hostname of the host.
	hostNameKey = "host.name"
	// `host.id` is a Unique host id.
	// As hostname is not always unique, use values that are meaningful in your environment.
	hostIDKey = "host.id"
	// `host.ip` is Host ip addresses.
	// Note: this field should contain an array of values.
	hostIPKey = "host.ip"
	// `host.mac` is Host mac addresses.
	// Note: this field should contain an array of values.
	hostMACKey = "host.mac"
)

// ECSMetadata returns an agent ECS compliant metadata.
func (i *AgentInfo) ECSMetadata() (map[string]interface{}, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	// TODO: remove these values when kibana migrates to ECS
	meta := map[string]interface{}{
		"platform": runtime.GOOS,
		"version":  release.Version(),
		"host":     hostname,
	}

	sysInfo, err := sysinfo.Host()
	if err != nil {
		return nil, err
	}

	info := sysInfo.Info()

	// Agent
	meta[agentIDKey] = i.agentID
	meta[agentVersionKey] = release.Version()

	// Host
	meta[hostArchKey] = info.Architecture
	meta[hostHostnameKey] = hostname
	meta[hostNameKey] = hostname
	meta[hostIDKey] = info.UniqueID
	meta[hostIPKey] = fmt.Sprintf("[%s]", strings.Join(info.IPs, ","))
	meta[hostMACKey] = fmt.Sprintf("[%s]", strings.Join(info.MACs, ","))

	// Operating system
	meta[osFamilyKey] = runtime.GOOS
	meta[osKernelKey] = info.KernelVersion
	meta[osPlatformKey] = info.OS.Family
	meta[osVersionKey] = info.OS.Version
	meta[osNameKey] = info.OS.Name
	meta[osFullKey] = getFullOSName(info)

	return meta, nil
}

func getFullOSName(info types.HostInfo) string {
	var sb strings.Builder
	sb.WriteString(info.OS.Name)
	if codeName := info.OS.Codename; codeName != "" {
		sb.WriteString(" ")
		sb.WriteString(codeName)
	}

	if version := info.OS.Version; version != "" {
		sb.WriteString("(")
		sb.WriteString(version)
		sb.WriteString(")")
	}

	return sb.String()
}
