// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package state

import (
	"net"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/dns"
)

type DNSTracker struct {
	// map[net.UDPAddr(string)][]dns.Transaction
	transactionByClient *common.Cache

	// map[net.UDPAddr(string)]*process
	processByClient *common.Cache
}

func NewDNSTracker(timeout time.Duration) *DNSTracker {
	return &DNSTracker{
		transactionByClient: common.NewCache(timeout, 8),
		processByClient:     common.NewCache(timeout, 8),
	}
}

// AddTransaction registers a new DNS transaction.
func (dt *DNSTracker) AddTransaction(tr dns.Transaction) {
	clientAddr := tr.Client.String()
	if procIf := dt.processByClient.Get(clientAddr); procIf != nil {
		if proc, ok := procIf.(*Process); ok {
			proc.addTransaction(tr)
			return
		}
	}
	var list []dns.Transaction
	if prev := dt.transactionByClient.Get(clientAddr); prev != nil {
		list = prev.([]dns.Transaction)
	}
	list = append(list, tr)
	dt.transactionByClient.Put(clientAddr, list)
}

// AddTransactionWithProcess registers a new DNS transaction for the given process.
func (dt *DNSTracker) AddTransactionWithProcess(tr dns.Transaction, proc *Process) {
	proc.addTransaction(tr)
}

// CleanUp removes expired entries from the maps.
func (dt *DNSTracker) CleanUp() {
	dt.transactionByClient.CleanUp()
	dt.processByClient.CleanUp()
}

// RegisterEndpoint registers a new local endpoint used for DNS queries
// to correlate captured DNS packets with their originator process.
func (dt *DNSTracker) RegisterEndpoint(addr net.UDPAddr, proc *Process) {
	key := addr.String()
	dt.processByClient.Put(key, proc)
	if listIf := dt.transactionByClient.Get(key); listIf != nil {
		list := listIf.([]dns.Transaction)
		for _, tr := range list {
			proc.addTransaction(tr)
		}
	}
}
