// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package common

import (
	"net"
	"time"

	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/dns"
)

type Process struct {
	PID                  uint32
	Name, Path           string
	Args                 []string
	created              uint64
	uid, gid, euid, egid uint32
	hasCreds             bool

	// populated by state from created
	createdTime time.Time

	// populated by DNS enrichment.
	resolvedDomains map[string]string
}

func CreateProcess(pid uint32, path, name string, created uint64, args []string) *Process {
	return &Process{
		PID:     pid,
		Path:    path,
		Args:    args,
		Name:    name,
		created: created,
	}
}

func (p *Process) SetCreds(uid, gid, euid, egid uint32) *Process {
	p.hasCreds = true
	p.uid = uid
	p.gid = gid
	p.euid = euid
	p.egid = egid
	return p
}

func (p *Process) SetCreated(ts time.Time) *Process {
	p.createdTime = ts
	return p
}

func (p *Process) FormatCreatedIfZero(formatter func(uint64) time.Time) *Process {
	if p.createdTime == (time.Time{}) {
		p.createdTime = formatter(p.created)
	}
	return p
}

// ResolveIP returns the domain associated with the given IP.
func (p *Process) ResolveIP(ip net.IP) (domain string, found bool) {
	domain, found = p.resolvedDomains[ip.String()]
	return
}

func (p *Process) AddTransaction(tr dns.Transaction) {
	if p.resolvedDomains == nil {
		p.resolvedDomains = make(map[string]string)
	}
	for _, addr := range tr.Addresses {
		p.resolvedDomains[addr.String()] = tr.Domain
	}
}
