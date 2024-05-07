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

//go:build linux

package network

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/gopacket/afpacket"
	psutil "github.com/shirou/gopsutil/process"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos/applayer"
	"github.com/elastic/elastic-agent-libs/logp"
)

// give the update/request channels a bit of a buffer,
// in cases where we're getting flooding with events and don't want to block.
const channelBaseSize = 10

// NewNetworkTracker creates a new network tracker for the given config
func NewNetworkTracker() (*Tracker, error) {
	watcher := &procs.ProcessesWatcher{}
	err := watcher.Init(procs.ProcsConfig{Enabled: true})
	if err != nil {
		return nil, fmt.Errorf("error initializing process watcher: %w", err)
	}

	tracker := &Tracker{
		procData: map[int]PacketData{},
		dataMut:  sync.RWMutex{},

		updateChan: make(chan counterUpdateEvent, channelBaseSize),
		reqChan:    make(chan requestCounters, channelBaseSize),
		stopChan:   make(chan struct{}, 1),
		// we don't want a garbage collection sweep that's too frequent,
		// as we might delete a short-lived process before system/process has even had a chance to report it
		// perhaps this should be `period`*2?
		gctime: time.Minute * 10,
		// right now, the packetbeat watcher won't work with alternate mountpoints,
		// as support is missing from go-sysinfo. This means we can't support /hostfs settings
		procWatcher: watcher,
		gcPIDFetch:  psutil.PidExistsWithContext,

		log: logp.L(),
	}

	return tracker, nil
}

// Update the tracker with the given counts
func (track *Tracker) Update(packetLen int, proto applayer.Transport, proc *common.ProcessTuple) {
	track.updateChan <- counterUpdateEvent{pktLen: packetLen, TransProtocol: proto, Proc: proc}
}

// Get data for a given PID
func (track *Tracker) Get(pid int) PacketData {
	req := requestCounters{PID: pid, Resp: make(chan PacketData)}
	track.reqChan <- req
	got := <-req.Resp
	return got
}

// Stop the tracker
func (track *Tracker) Stop() {
	track.stopChan <- struct{}{}
}

// Track is a non-blocking operation that starts a packet sniffer, and the underlying
// tracker that correlates packet data with pids
func (track *Tracker) Track() error {
	var afHandle *afpacket.TPacket
	var err error
	if !track.testmode {
		afHandle, err = afpacket.NewTPacket(afpacket.SocketRaw)
		if err != nil {
			return fmt.Errorf("error creating afpacket interface: %w", err)
		}
	}

	helperContext, helperCancel := context.WithCancel(context.Background())
	go func() {
		if !track.testmode {
			err := runPacketHandle(helperContext, afHandle, track.procWatcher, track)
			if err != nil {
				track.log.Errorf("error starting packet capture: %s", err)
				helperCancel()
			}
		}
	}()

	go func() {
		track.garbageCollect(helperContext)
	}()

	go func() {
		for {
			select {
			case update := <-track.updateChan:
				track.updateInternalTracking(update)
			case req := <-track.reqChan:
				track.dataMut.RLock()
				proc, ok := track.procData[req.PID]
				track.dataMut.RUnlock()

				if ok {
					req.Resp <- proc
				} else {
					req.Resp <- PacketData{}
				}
			case <-helperContext.Done():
				return
			case <-track.stopChan:
				helperCancel()
				return
			}
		}
	}()

	return nil
}

// As far as I can see, the only reliable way to do garbage collection
// is to check each pid on a timer. Simply relying on a timestamp or counter could
// result in false positives for PIDs that only occasionally create network traffic.
// This method operates in its own thread, and will check the list of known PIDs on a timer.
func (track *Tracker) garbageCollect(ctx context.Context) {
	// the timer should be set to a fairly large number; if we garbage-collect more frequently there's
	// a risk that we clean up a dead PID before its been reported.
	ticker := time.NewTicker(track.gctime)
	for {
		select {
		case <-ticker.C:
			//copy total proc list so we don't hold the mutex for any longer than needed
			track.dataMut.RLock()
			keys := make([]int, 0, len(track.procData))
			for k := range track.procData {
				keys = append(keys, k)
			}
			track.dataMut.RUnlock()
			keysToDelete := []int{}
			for _, key := range keys {
				found, _ := track.gcPIDFetch(ctx, int32(key))
				if !found {
					keysToDelete = append(keysToDelete, key)
				}
			}
			if len(keysToDelete) > 0 {
				track.dataMut.Lock()
				for _, key := range keysToDelete {
					delete(track.procData, key)
				}
				track.dataMut.Unlock()
				track.log.Debugf("removed PIDs %v from network process tracker", keysToDelete)
			}
		case <-ctx.Done():
			return
		}

		// used to coordinate testing, not used in prod
		if track.loopWaiter != nil {
			track.loopWaiter <- struct{}{}
		}

	}

}

func (track *Tracker) updateInternalTracking(update counterUpdateEvent) {
	track.dataMut.Lock()
	defer track.dataMut.Unlock()
	if update.Proc.Src.PID != 0 {
		track.updateOutgoing(update.TransProtocol, update.Proc, update.pktLen)
	}
	if update.Proc.Dst.PID != 0 {
		track.updateIncoming(update.TransProtocol, update.Proc, update.pktLen)
	}
}

func (track *Tracker) updateOutgoing(proto applayer.Transport, proc *common.ProcessTuple, pktLen int) {
	newPort := PacketData{}
	if port, ok := track.procData[proc.Src.PID]; ok {
		newPort = port
	}

	if proto == applayer.TransportTCP {
		newPort.Outgoing.TCP = newPort.Outgoing.TCP + uint64(pktLen)
	} else if proto == applayer.TransportUDP {
		newPort.Outgoing.UDP = newPort.Outgoing.UDP + uint64(pktLen)
	}

	track.procData[proc.Src.PID] = newPort
}

func (track *Tracker) updateIncoming(proto applayer.Transport, proc *common.ProcessTuple, pktLen int) {
	newPort := PacketData{}
	if port, ok := track.procData[proc.Dst.PID]; ok {
		newPort = port
	}

	if proto == applayer.TransportTCP {
		newPort.Incoming.TCP = newPort.Incoming.TCP + uint64(pktLen)
	} else if proto == applayer.TransportUDP {
		newPort.Incoming.UDP = newPort.Incoming.UDP + uint64(pktLen)

	}

	track.procData[proc.Dst.PID] = newPort
}
