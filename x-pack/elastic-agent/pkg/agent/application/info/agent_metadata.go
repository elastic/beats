// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package info

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
	"github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"
)

// ECSMeta is a collection of agent related metadata in ECS compliant object form.
type ECSMeta struct {
	Elastic *ElasticECSMeta `json:"elastic"`
	Host    *HostECSMeta    `json:"host"`
	OS      *SystemECSMeta  `json:"os"`
}

// ElasticECSMeta is a collection of elastic vendor metadata in ECS compliant object form.
type ElasticECSMeta struct {
	Agent *AgentECSMeta `json:"agent"`
}

// AgentECSMeta is a collection of agent metadata in ECS compliant object form.
type AgentECSMeta struct {
	// ID is a generated (in standalone) or assigned (in fleet) agent identifier.
	ID string `json:"id"`
	// Version specifies current version of an agent.
	Version string `json:"version"`
	// Snapshot is a flag specifying that the agent used is a snapshot build.
	Snapshot bool `json:"snapshot"`
	// BuildOriginal is an extended build information for the agent.
	BuildOriginal string `json:"build.original"`
	// Upgradeable is a flag specifying if it is possible for agent to be upgraded.
	Upgradeable bool `json:"upgradeable"`
	// LogLevel describes currently set log level.
	// Possible values: "debug"|"info"|"warning"|"error"
	LogLevel string `json:"log_level"`
}

// SystemECSMeta is a collection of operating system metadata in ECS compliant object form.
type SystemECSMeta struct {
	// Family defines a family of underlying operating system (e.g. redhat, debian, freebsd, windows).
	Family string `json:"family"`
	// Kernel specifies current version of a kernel in a semver format.
	Kernel string `json:"kernel"`
	// Platform specifies platform agent is running on (e.g. centos, ubuntu, windows).
	Platform string `json:"platform"`
	// Version specifies version of underlying operating system (e.g. 10.12.6).
	Version string `json:"version"`
	// Name is a operating system name.
	// Currently we just normalize the name (i.e. macOS, Windows, Linux). See https://www.elastic.co/guide/en/ecs/current/ecs-html
	Name string `json:"name"`
	// Full is an operating system name, including the version or code name.
	FullName string `json:"full"`
}

// HostECSMeta is a collection of host metadata in ECS compliant object form.
type HostECSMeta struct {
	// Arch defines architecture of a host (e.g. x86_64, arm, ppc, mips).
	Arch string `json:"architecture"`
	// Hostname specifies hostname of the host.
	Hostname string `json:"hostname"`
	// Name specifies hostname of the host.
	Name string `json:"name"`
	// ID is a Unique host id.
	// As hostname is not always unique, use values that are meaningful in your environment.
	ID string `json:"id"`
	// IP is Host ip addresses.
	// Note: this field should contain an array of values.
	IP []string `json:"ip"`
	// Mac is Host mac addresses.
	// Note: this field should contain an array of values.
	MAC []string `json:"mac"`
}

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

// Metadata loads metadata from disk.
func Metadata() (*ECSMeta, error) {
	agentInfo, err := NewAgentInfo(false)
	if err != nil {
		return nil, err
	}

	meta, err := agentInfo.ECSMetadata()
	if err != nil {
		return nil, errors.New(err, "failed to gather host metadata")
	}

	return meta, nil
}

// ECSMetadata returns an agent ECS compliant metadata.
func (i *AgentInfo) ECSMetadata() (*ECSMeta, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	sysInfo, err := sysinfo.Host()
	if err != nil {
		return nil, err
	}

	info := sysInfo.Info()

	return &ECSMeta{
		Elastic: &ElasticECSMeta{
			Agent: &AgentECSMeta{
				ID:            i.agentID,
				Version:       release.Version(),
				Snapshot:      release.Snapshot(),
				BuildOriginal: release.Info().String(),
				// only upgradeable if running from Agent installer and running under the
				// control of the system supervisor (or built specifically with upgrading enabled)
				Upgradeable: release.Upgradeable() || (RunningInstalled() && RunningUnderSupervisor()),
				LogLevel:    i.LogLevel(),
			},
		},
		Host: &HostECSMeta{
			Arch:     info.Architecture,
			Hostname: hostname,
			Name:     hostname,
			ID:       info.UniqueID,
			IP:       info.IPs,
			MAC:      info.MACs,
		},

		// Operating system
		OS: &SystemECSMeta{
			Family:   info.OS.Family,
			Kernel:   info.KernelVersion,
			Platform: info.OS.Platform,
			Version:  info.OS.Version,
			Name:     info.OS.Name,
			FullName: getFullOSName(info),
		},
	}, nil
}

// ECSMetadataFlatMap returns an agent ECS compliant metadata in a form of flattened map.
func (i *AgentInfo) ECSMetadataFlatMap() (map[string]interface{}, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	// TODO: remove these values when kibana migrates to ECS
	meta := make(map[string]interface{})

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
