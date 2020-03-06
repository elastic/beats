// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package state

import (
	"net"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	socket_common "github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/common"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/dns"
)

type dnsTracker struct {
	// map[net.UDPAddr(string)][]dns.Transaction
	transactionByClient *common.Cache

	// map[net.UDPAddr(string)]*process
	processByClient *common.Cache
}

func newDNSTracker(timeout time.Duration) *dnsTracker {
	return &dnsTracker{
		transactionByClient: common.NewCache(timeout, 8),
		processByClient:     common.NewCache(timeout, 8),
	}
}

// addTransaction registers a new DNS transaction.
func (dt *dnsTracker) addTransaction(tr dns.Transaction) {
	clientAddr := tr.Client.String()
	if procIf := dt.processByClient.Get(clientAddr); procIf != nil {
		if proc, ok := procIf.(*socket_common.Process); ok {
			proc.AddTransaction(tr)
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
func (dt *dnsTracker) addTransactionWithProcess(tr dns.Transaction, proc *socket_common.Process) {
	proc.AddTransaction(tr)
}

// CleanUp removes expired entries from the maps.
func (dt *dnsTracker) cleanUp() {
	dt.transactionByClient.CleanUp()
	dt.processByClient.CleanUp()
}

// RegisterEndpoint registers a new local endpoint used for DNS queries
// to correlate captured DNS packets with their originator process.
func (dt *dnsTracker) registerEndpoint(addr *net.UDPAddr, proc *socket_common.Process) {
	key := addr.String()
	dt.processByClient.Put(key, proc)
	if listIf := dt.transactionByClient.Get(key); listIf != nil {
		list := listIf.([]dns.Transaction)
		for _, tr := range list {
			proc.AddTransaction(tr)
		}
	}
}
