// +build linux

package socket

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/beats/metricbeat/module/system"
	"github.com/elastic/gosigar/sys/linux"

	"github.com/pkg/errors"
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

type MetricSet struct {
	mb.BaseMetricSet
	readBuffer    []byte
	seq           uint32
	ptable        *ProcTable
	euid          int
	previousConns hashSet
	currentConns  hashSet
	reverseLookup *ReverseLookupCache
	listeners     *ListenerTable
	users         UserCache
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The system collector metricset is beta")

	c := defaultConfig
	if err := base.Module().UnpackConfig(&c); err != nil {
		return nil, err
	}

	systemModule, ok := base.Module().(*system.Module)
	if !ok {
		return nil, errors.New("unexpected module type")
	}

	ptable, err := NewProcTable(filepath.Join(systemModule.HostFS, "/proc"))
	if err != nil {
		return nil, err
	}
	if os.Geteuid() != 0 {
		logp.Info("socket process info will only be available for " +
			"metricbeat because the process is running as a non-root user")
	}

	m := &MetricSet{
		BaseMetricSet: base,
		readBuffer:    make([]byte, os.Getpagesize()),
		ptable:        ptable,
		euid:          os.Geteuid(),
		previousConns: hashSet{},
		currentConns:  hashSet{},
		listeners:     NewListenerTable(),
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

func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	// Refresh inode to process mapping (must be root).
	if err := m.ptable.Refresh(); err != nil {
		debugf("process table refresh had failures: %v", err)
	}

	// Send request over netlink and parse responses.
	req := linux.NewInetDiagReq()
	req.Header.Seq = atomic.AddUint32(&m.seq, 1)
	sockets, err := linux.NetlinkInetDiagWithBuf(req, m.readBuffer, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed requesting socket dump")
	}
	debugf("netlink returned %d sockets", len(sockets))

	// Filter sockets that were known during the previous poll.
	sockets = m.filterAndRememberSockets(sockets)

	// Enrich sockets with direction/pid/process/user/hostname and convert to MapStr.
	rtn := make([]common.MapStr, 0, len(sockets))
	for _, s := range sockets {
		c := newConnection(s)
		m.enrichConnectionData(c)
		rtn = append(rtn, c.ToMapStr())
	}

	// Set the "previous" connections set to the "current" connections.
	tmp := m.previousConns
	m.previousConns = m.currentConns
	m.currentConns = tmp.Reset()

	// Reset the listeners for the next iteration.
	m.listeners.Reset()

	return rtn, nil
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
	c.Username = m.users.LookupUID(int(c.UID))

	// Determine direction (incoming, outgoing, or listening).
	c.Direction = m.listeners.Direction(uint8(syscall.IPPROTO_TCP),
		c.LocalIP, c.LocalPort, c.RemoteIP, c.RemotePort)

	// Reverse DNS lookup on the remote IP.
	if m.reverseLookup != nil && c.Direction != Listening {
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
	} else if m.euid == 0 {
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
	Direction Direction

	DestHost            string // Reverse lookup of dest IP.
	DestHostETLDPlusOne string
	DestHostError       error // Resolver error.

	// Process identifiers.
	Inode        uint32 // Inode of the socket.
	PID          int    // PID of the socket owner.
	Exe          string // Absolute path to the executable.
	Command      string // Command
	CmdLine      string // Full command line with arguments.
	ProcessError error  // Reason process info is unavailable.

	// User identifiers.
	UID      uint32 // UID of the socket owner.
	Username string // Username of the socket.
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

func (c *connection) ToMapStr() common.MapStr {
	evt := common.MapStr{
		"family": c.Family.String(),
		"local": common.MapStr{
			"ip":   c.LocalIP.String(),
			"port": c.LocalPort,
		},
		"user": common.MapStr{
			"id": c.UID,
		},
		"direction": c.Direction.String(),
	}

	if c.Username != "" {
		evt.Put("user.name", c.Username)
	}

	if c.ProcessError != nil {
		evt.Put("process.error", c.ProcessError.Error())
	} else {
		process := common.MapStr{"pid": c.PID}
		evt["process"] = process

		if c.PID > 0 {
			addOptionalString(process, "exe", c.Exe)
			addOptionalString(process, "command", c.Command)
			addOptionalString(process, "cmdline", c.CmdLine)
		} else if c.PID == 0 {
			process["command"] = "kernel"
		}
	}

	if c.RemotePort != 0 {
		remote := common.MapStr{
			"ip":   c.RemoteIP.String(),
			"port": c.RemotePort,
		}
		evt["remote"] = remote

		if c.DestHostError != nil {
			remote["host_error"] = c.DestHostError.Error()
		} else {
			addOptionalString(remote, "host", c.DestHost)
			addOptionalString(remote, "etld_plus_one", c.DestHostETLDPlusOne)
		}
	}

	return evt
}

func addOptionalString(m common.MapStr, key, value string) {
	if value == "" {
		return
	}
	m[key] = value
}
