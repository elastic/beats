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
// +build linux

package socket

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"strconv"
	"syscall"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	sock "github.com/elastic/beats/v7/metricbeat/helper/socket"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/gosigar/sys/linux"
)

var (
	debugSelector = "system.socket"
	debugf        = logp.MakeDebug(debugSelector)
)

func init() {
	mb.Registry.MustAddMetricSet("system", "socket", New,
		mb.WithHostParser(parse.EmptyHostParser),
	)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	netlink       *sock.NetlinkSession
	ptable        *sock.ProcTable
	euid          int
	previousConns hashSet
	currentConns  hashSet
	reverseLookup *ReverseLookupCache
	listeners     *sock.ListenerTable
	users         UserCache
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	sys := base.Module().(resolve.Resolver)
	c := defaultConfig
	if err := base.Module().UnpackConfig(&c); err != nil {
		return nil, err
	}

	ptable, err := sock.NewProcTable(sys.ResolveHostFS("/proc"))
	if err != nil {
		return nil, err
	}
	if !ptable.Privileged() {
		logp.Info("socket process info will only be available for processes owned by the %v user "+
			"because this Beat is not running with enough privileges", os.Geteuid())
	}

	m := &MetricSet{
		BaseMetricSet: base,
		netlink:       sock.NewNetlinkSession(),
		ptable:        ptable,
		euid:          os.Geteuid(),
		previousConns: hashSet{},
		currentConns:  hashSet{},
		listeners:     sock.NewListenerTable(),
		users:         NewUserCache(),
	}

	if c.ReverseLookup.IsEnabled() {
		var successTTL, failureTTL = defSuccessTTL, defFailureTTL
		if c.ReverseLookup.SuccessTTL != 0 {
			successTTL = c.ReverseLookup.SuccessTTL
		}
		if c.ReverseLookup.FailureTTL != 0 {
			successTTL = c.ReverseLookup.FailureTTL
		}
		debugf("enabled reverse DNS lookup with cache TTL of %v/%v",
			successTTL, failureTTL)
		m.reverseLookup = NewReverseLookupCache(successTTL, failureTTL)
	}

	return m, nil
}

// Fetch socket metrics from the system
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	// Refresh inode to process mapping (must be root).
	if err := m.ptable.Refresh(); err != nil {
		debugf("process table refresh had failures: %v", err)
	}

	sockets, err := m.netlink.GetSocketList()
	if err != nil {
		return errors.Wrap(err, "failed requesting socket dump")
	}
	debugf("netlink returned %d sockets", len(sockets))

	// Filter sockets that were known during the previous poll.
	sockets = m.filterAndRememberSockets(sockets)

	// Enrich sockets with direction/pid/process/user/hostname and convert to MapStr.
	for _, s := range sockets {
		c := newConnection(s)
		m.enrichConnectionData(c)

		root, metricSet := c.ToMapStr()

		isOpen := r.Event(mb.Event{
			RootFields:      root,
			MetricSetFields: metricSet,
		})
		if !isOpen {
			return nil
		}
	}

	// Set the "previous" connections set to the "current" connections.
	tmp := m.previousConns
	m.previousConns = m.currentConns
	m.currentConns = tmp.Reset()

	// Reset the listeners for the next iteration.
	m.listeners.Reset()

	return nil
}

// filterAndRememberSockets filters sockets to remove sockets that were seen
// during the last poll. It stores all of the sockets it sees for the next
// poll.
func (m *MetricSet) filterAndRememberSockets(sockets ...[]*linux.InetDiagMsg) []*linux.InetDiagMsg {
	var newSockets []*linux.InetDiagMsg
	for _, list := range sockets {
		for _, socket := range list {
			// Register all listening sockets.
			if socket.DstPort() == 0 {
				m.listeners.Put(uint8(syscall.IPPROTO_TCP), socket.SrcIP(), socket.SrcPort())
			}

			// Filter known sockets.
			if m.isNewSocket(socket) {
				if logp.IsDebug(debugSelector) {
					debugf("found new socket %v:%v -> %v:%v with state=%v, inode=%v, hash-id=%d",
						socket.SrcIP(), socket.SrcPort(),
						socket.DstIP(), socket.DstPort(),
						linux.TCPState(socket.State), socket.Inode, socket.FastHash())
				}
				newSockets = append(newSockets, socket)
			}
		}
	}
	return newSockets
}

// isNewSocket returns true if the socket is new since the last poll.
func (m *MetricSet) isNewSocket(diag *linux.InetDiagMsg) bool {
	// Don't use the socket's inode for deduplication because once the socket
	// is closing the inode goes to 0.
	key := diag.FastHash()
	m.currentConns.Add(key)
	return !m.previousConns.Contains(key)
}

// enrichConnectionData enriches the connection with username, direction,
// hostname of the remote IP (if enabled), eTLD + 1 of the hostname, and the
// process owning the socket.
func (m *MetricSet) enrichConnectionData(c *connection) {
	c.User = m.users.LookupUID(int(c.UID))

	// Determine direction (incoming, outgoing, or listening).
	c.Direction = m.listeners.Direction(uint8(c.Family), uint8(syscall.IPPROTO_TCP),
		c.LocalIP, c.LocalPort, c.RemoteIP, c.RemotePort)

	// Reverse DNS lookup on the remote IP.
	if m.reverseLookup != nil && c.Direction != sock.Listening {
		hostname, err := m.reverseLookup.Lookup(c.RemoteIP)
		if err != nil {
			c.DestHostError = err
		} else {
			c.DestHost = hostname
			c.DestHostETLDPlusOne, _ = etldPlusOne(hostname)
		}
	}

	// Add process info by finding the process that holds the socket's inode.
	if proc := m.ptable.ProcessBySocketInode(c.Inode); proc != nil {
		c.PID = proc.PID
		c.Exe = proc.Executable
		c.Command = proc.Command
		c.CmdLine = proc.CmdLine
		c.Args = proc.Args
	} else if m.ptable.Privileged() {
		if c.Inode == 0 {
			c.ProcessError = fmt.Errorf("process has exited. inode=%v, tcp_state=%v",
				c.Inode, c.State)
		} else {
			c.ProcessError = fmt.Errorf("process not found. inode=%v, tcp_state=%v",
				c.Inode, c.State)
		}
	}
}

type connection struct {
	Family     linux.AddressFamily
	LocalIP    net.IP
	LocalPort  int
	RemoteIP   net.IP
	RemotePort int

	State     linux.TCPState
	Direction sock.Direction

	DestHost            string // Reverse lookup of dest IP.
	DestHostETLDPlusOne string
	DestHostError       error // Resolver error.

	// Process identifiers.
	Inode        uint32   // Inode of the socket.
	PID          int      // PID of the socket owner.
	Exe          string   // Absolute path to the executable.
	Command      string   // Command
	CmdLine      string   // Full command line with arguments.
	Args         []string // Raw arguments
	ProcessError error    // Reason process info is unavailable.

	// User identifiers.
	UID  uint32     // UID of the socket owner.
	User *user.User // Owner of the socket.
}

func newConnection(diag *linux.InetDiagMsg) *connection {
	return &connection{
		Family:     linux.AddressFamily(diag.Family),
		State:      linux.TCPState(diag.State),
		LocalIP:    diag.SrcIP(),
		LocalPort:  diag.SrcPort(),
		RemoteIP:   diag.DstIP(),
		RemotePort: diag.DstPort(),
		Inode:      diag.Inode,
		UID:        diag.UID,
		PID:        -1,
	}
}

// Map helpers for conversion to event
var (
	ianaNumbersMap = map[string]string{
		"ipv4": "4",
		"ipv6": "41",
	}

	localHostInfoGroup = map[string]string{
		sock.IngressName:   "destination",
		sock.EgressName:    "source",
		sock.ListeningName: "server",
	}

	remoteHostInfoGroup = map[string]string{
		sock.IngressName: "source",
		sock.EgressName:  "destination",
	}
)

func (c *connection) ToMapStr() (fields common.MapStr, metricSetFields common.MapStr) {
	localGroup := "server"
	if g, ok := localHostInfoGroup[c.Direction.String()]; ok {
		localGroup = g
	}

	fields = common.MapStr{
		"network": common.MapStr{
			"type":        c.Family.String(),
			"iana_number": ianaNumbersMap[c.Family.String()],
			"direction":   c.Direction.String(),
		},
		"user": common.MapStr{
			"id": strconv.Itoa(int(c.UID)),
		},
		// Aliases for this are not going to be possible, keeping
		// duplicated fields by now for backwards comatibility
		localGroup: common.MapStr{
			"ip":   c.LocalIP.String(),
			"port": c.LocalPort,
		},
	}

	metricSetFields = common.MapStr{
		"local": common.MapStr{
			"ip":   c.LocalIP.String(),
			"port": c.LocalPort,
		},
	}

	if c.User.Username != "" {
		fields.Put("user.name", c.User.Username)
	}

	if c.User.Name != "" {
		fields.Put("user.full_name", c.User.Name)
	}

	if c.ProcessError != nil {
		fields.Put("error.code", c.ProcessError.Error())
	} else {
		process := common.MapStr{"pid": c.PID}

		if c.PID > 0 {
			addOptionalString(process, "executable", c.Exe)
			addOptionalString(process, "name", c.Command)

			if len(c.Args) >= 0 {
				process["args"] = c.Args
				metricSetFields["process"] = common.MapStr{
					"cmdline": c.CmdLine,
				}
			}
		} else if c.PID == 0 {
			process["command"] = "kernel"
		}

		if c.PID >= 0 {
			fields["process"] = process
		}
	}

	if c.RemotePort != 0 {
		// Aliases for this are not going to be possible, keeping
		// duplicated fields by now for backwards comatibility
		remote := common.MapStr{
			"ip":   c.RemoteIP.String(),
			"port": c.RemotePort,
		}
		if c.DestHostError != nil {
			remote["host_error"] = c.DestHostError.Error()
		} else {
			addOptionalString(remote, "host", c.DestHost)
			addOptionalString(remote, "etld_plus_one", c.DestHostETLDPlusOne)
		}
		metricSetFields["remote"] = remote

		remoteGroup, ok := remoteHostInfoGroup[c.Direction.String()]
		if ok {
			fields[remoteGroup] = common.MapStr{
				"ip":   c.RemoteIP.String(),
				"port": c.RemotePort,
			}
		}
	}

	return fields, metricSetFields
}

func addOptionalString(m common.MapStr, key, value string) {
	if value == "" {
		return
	}
	m[key] = value
}
