// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package info

import (
	"os"
	"runtime"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
	"github.com/elastic/go-sysinfo"
)

// List of variables available to be used in constraint definitions.
const (
	// `agent.id` is a generated (in standalone) or assigned (in fleet) agent identifier.
	agentIDKey = "agent.id"
	// `agent.version` specifies current version of an agent.
	agentVersionKey = "agent.version"
	// `host.architecture` defines architecture of a host (e.g. x86_64, arm, ppc, mips).
	hostArchKey = "host.architecture"
	// `os.family` defines a family of underlying operating system (e.g. redhat, debian, freebsd, windows).
	osFamilyKey = "os.family"
	// `os.kernel` specifies current version of a kernel in a semver format.
	osKernelKey = "os.kernel"
	// `os.platform` specifies platform agent is running on (e.g. centos, ubuntu, windows).
	osPlatformKey = "os.platform"
	// `os.version` specifies version of underlying operating system (e.g. 10.12.6).
	osVersionKey = "os.version"
	// `host.hostname` specifies hostname of the host.
	hostHostnameKey = "host.hostname"
	// `host.name` specifies hostname of the host.
	hostNameKey = "host.name"
)

// AgentID returns an agent identifier.
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

	// 	Agent
	meta[agentIDKey] = i.agentID
	meta[agentVersionKey] = release.Version()

	// Host
	meta[hostArchKey] = info.Architecture
	meta[hostHostnameKey] = hostname
	meta[hostNameKey] = hostname

	// Operating system
	meta[osFamilyKey] = runtime.GOOS
	meta[osKernelKey] = info.KernelVersion
	meta[osPlatformKey] = info.OS.Family
	meta[osVersionKey] = info.OS.Version

	return meta, nil
}
