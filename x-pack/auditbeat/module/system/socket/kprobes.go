// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build (linux && 386) || (linux && amd64)
// +build linux,386 linux,amd64

package socket

import (
	"errors"
	"fmt"
	"strings"
	"unsafe"

	"github.com/joeshaw/multierror"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/x-pack/auditbeat/module/system/socket/helper"
	"github.com/elastic/beats/v8/x-pack/auditbeat/tracing"
)

// This is how many data we dump from sk_buff->data to read full packet headers
// (IP + UDP header). This has been observed to include up to 100 bytes of
// padding.
const skBuffDataDumpBytes = 256

// ProbeTransform transforms a probe before its installed.
type ProbeTransform func(helper.ProbeDef) helper.ProbeDef

// ProbeInstaller installs and uninstalls probes.
type probeInstaller struct {
	traceFS    *tracing.TraceFS
	transforms []ProbeTransform
	installed  []tracing.Probe
}

func newProbeInstaller(tfs *tracing.TraceFS, transforms ...ProbeTransform) helper.ProbeInstaller {
	return &probeInstaller{
		traceFS:    tfs,
		transforms: transforms,
	}
}

// Install the given probe.
func (p *probeInstaller) Install(pdef helper.ProbeDef) (format tracing.ProbeFormat, decoder tracing.Decoder, err error) {
	for _, d := range p.transforms {
		pdef = d(pdef)
	}
	if pdef.Decoder == nil {
		return format, decoder, errors.New("nil decoder in probe definition")
	}
	if err = p.traceFS.AddKProbe(pdef.Probe); err != nil {
		return format, decoder, fmt.Errorf("failed installing probe '%s': %w", pdef.Probe.String(), err)
	}
	p.installed = append(p.installed, pdef.Probe)
	if format, err = p.traceFS.LoadProbeFormat(pdef.Probe); err != nil {
		return format, decoder, fmt.Errorf("failed to load probe format: %w", err)
	}
	if decoder, err = pdef.Decoder(format); err != nil {
		return format, decoder, fmt.Errorf("failed to create decoder: %w", err)
	}
	return
}

// UninstallInstalled uninstalls the probes installed by Install.
func (p *probeInstaller) UninstallInstalled() error {
	var errs multierror.Errors
	for _, probe := range p.installed {
		if err := p.traceFS.RemoveKProbe(probe); err != nil {
			errs = append(errs, err)
		}
	}
	p.installed = nil
	return errs.Err()
}

// UninstallIf uninstalls all probes in the system that met the condition.
func (p *probeInstaller) UninstallIf(condition helper.ProbeCondition) error {
	kprobes, err := p.traceFS.ListKProbes()
	if err != nil {
		return fmt.Errorf("failed to list installed kprobes: %w", err)
	}
	var errs multierror.Errors
	for _, probe := range kprobes {
		if condition(probe) {
			if err := p.traceFS.RemoveKProbe(probe); err != nil {
				errs = append(errs, fmt.Errorf("unable to remove kprobe '%s': %w", probe.String(), err))
			}
		}
	}
	return errs.Err()
}

// WithGroup sets a custom group to probes before they are installed.
func WithGroup(name string) ProbeTransform {
	return func(probe helper.ProbeDef) helper.ProbeDef {
		probe.Probe.Group = name
		return probe
	}
}

// WithTemplates expands templates in probes before they are installed.
func WithTemplates(vars common.MapStr) ProbeTransform {
	return func(probe helper.ProbeDef) helper.ProbeDef {
		return probe.ApplyTemplate(vars)
	}
}

// WithNoOp is a no-op transform.
func WithNoOp() ProbeTransform {
	return func(def helper.ProbeDef) helper.ProbeDef {
		return def
	}
}

// WithFilterPort is used for filtering port 22 traffic when debugging over
// an SSH connection. Otherwise there is a feedback loop when tracing events
// printed on the terminal are transmitted over SSH, which causes more tracing
// events.
func WithFilterPort(portnum uint16) ProbeTransform {
	var buf [2]byte
	tracing.MachineEndian.PutUint16(buf[:], portnum)
	filter := fmt.Sprintf("lport!=0x%02x%02x", buf[0], buf[1])
	return func(probe helper.ProbeDef) helper.ProbeDef {
		if strings.Contains(probe.Probe.Fetchargs, "lport=") {
			if probe.Probe.Filter == "" {
				probe.Probe.Filter = filter
			} else {
				probe.Probe.Filter = fmt.Sprintf("%s && (%s)",
					filter, probe.Probe.Filter)
			}
		}
		return probe
	}
}

// KProbes shared with IPv4 and IPv6.
var sharedKProbes = []helper.ProbeDef{
	//***************************************************************************
	//* RUNNING PROCESSES
	//***************************************************************************

	{
		Probe: tracing.Probe{
			Name:    "sys_execve_call",
			Address: "{{.SYS_EXECVE}}",
			Fetchargs: fmt.Sprintf("path=%s argptrs=%s param0=%s param1=%s param2=%s param3=%s param4=%s",
				helper.MakeMemoryDump("{{.SYS_P1}}", 0, maxProgArgLen),                                  // path
				helper.MakeMemoryDump("{{.SYS_P2}}", 0, int((maxProgArgs+1)*unsafe.Sizeof(uintptr(0)))), // argptrs
				helper.MakeMemoryDump("+{{call .POINTER_INDEX 0}}({{.SYS_P2}})", 0, maxProgArgLen),      // param0
				helper.MakeMemoryDump("+{{call .POINTER_INDEX 1}}({{.SYS_P2}})", 0, maxProgArgLen),      // param1
				helper.MakeMemoryDump("+{{call .POINTER_INDEX 2}}({{.SYS_P2}})", 0, maxProgArgLen),      // param2
				helper.MakeMemoryDump("+{{call .POINTER_INDEX 3}}({{.SYS_P2}})", 0, maxProgArgLen),      // param3
				helper.MakeMemoryDump("+{{call .POINTER_INDEX 4}}({{.SYS_P2}})", 0, maxProgArgLen),      // param4
			),
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(execveCall) }),
	},

	{
		Probe: tracing.Probe{
			Type:      tracing.TypeKRetProbe,
			Name:      "sys_execve_ret",
			Address:   "{{.SYS_EXECVE}}",
			Fetchargs: "retval={{.RET}}:s32",
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(execveRet) }),
	},

	{
		Probe: tracing.Probe{
			Name:    "do_exit",
			Address: "do_exit",
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(doExit) }),
	},

	{
		Probe: tracing.Probe{
			Name:      "commit_creds",
			Address:   "commit_creds",
			Fetchargs: "uid=+{{.STRUCT_CRED_UID}}({{.P1}}):u32 gid=+{{.STRUCT_CRED_GID}}({{.P1}}):u32 euid=+{{.STRUCT_CRED_EUID}}({{.P1}}):u32 egid=+{{.STRUCT_CRED_EGID}}({{.P1}}):u32",
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(commitCreds) }),
	},

	{
		Probe: tracing.Probe{
			Type:      tracing.TypeKRetProbe,
			Name:      "clone3_ret",
			Address:   "{{.DO_FORK}}",
			Fetchargs: "retval={{.RET}}",
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(forkRet) }),
	},
	/***************************************************************************
	 * IPv4
	 **************************************************************************/

	{
		Probe: tracing.Probe{
			Name:      "sock_init_data",
			Address:   "sock_init_data",
			Fetchargs: "socket={{.P1}} sock={{.P2}}",
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(sockInitData) }),
	},

	// IPv4/TCP/UDP socket created. Good for associating sockets with pids.
	// ** This is a struct socket* not a struct sock* **
	//
	//  " inet_create(socket=0xffff9f1ddadb8080, proto=17) "
	{
		Probe: tracing.Probe{
			Name:      "inet_create",
			Address:   "inet_create",
			Fetchargs: "proto={{.P3}}:s32",
			// proto=0 will select the protocol by looking at socket type (STREAM|DGRAM)
			Filter: "proto==0 || proto=={{.IPPROTO_TCP}} || proto=={{.IPPROTO_UDP}}",
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(inetCreate) }),
	},

	// IPv4/TCP/UDP socket released. Good for associating sockets with pids.
	{
		Probe: tracing.Probe{
			Name:      "inet_release",
			Address:   "inet_release",
			Fetchargs: "sock=+{{.SOCKET_SOCK}}({{.P1}})",
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(inetReleaseCall) }),
	},

	/***************************************************************************
	 * IPv4 / TCP
	 **************************************************************************/

	// An IPv4 / TCP socket connect attempt:
	//
	//  " connect(sock=0xffff9f1ddd216040, 0.0.0.0:0 -> 151.101.66.217:443) "
	{
		Probe: tracing.Probe{
			Name:      "tcp4_connect_in",
			Address:   "tcp_v4_connect",
			Fetchargs: "sock={{.P1}} laddr=+{{.INET_SOCK_LADDR}}({{.P1}}):u32 lport=+{{.INET_SOCK_LPORT}}({{.P1}}):u16 af=+{{.SOCKADDR_IN_AF}}({{.P2}}):u16 addr=+{{.SOCKADDR_IN_ADDR}}({{.P2}}):u32 port=+{{.SOCKADDR_IN_PORT}}({{.P2}}):u16",
			Filter:    "af=={{.AF_INET}}",
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(tcpIPv4ConnectCall) }),
	},

	// Result of IPv4/TCP connect:
	//
	//  " <- connect ok (retval==0 or retval==-ERRNO) "
	{
		Probe: tracing.Probe{
			Type:      tracing.TypeKRetProbe,
			Name:      "tcp4_connect_out",
			Address:   "tcp_v4_connect",
			Fetchargs: "retval={{.RET}}:s32",
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(tcpConnectResult) }),
	},

	// IPv4 packet is sent. Acceptable as a packet counter,
	// But the actual data sent might span multiple packets if TSO is in use.
	//
	// (lport is fetched just for the sake of dev mode filtering).
	//
	//  " ip_local_out(sock=0xffff9f1ddd216040) "
	{
		Probe: tracing.Probe{
			Name:      "ip_local_out_call",
			Address:   "{{.IP_LOCAL_OUT}}",
			Fetchargs: "sock={{.IP_LOCAL_OUT_SOCK}} size=+{{.SK_BUFF_LEN}}({{.IP_LOCAL_OUT_SK_BUFF}}):u32 af=+{{.INET_SOCK_AF}}({{.IP_LOCAL_OUT_SOCK}}):u16 laddr=+{{.INET_SOCK_LADDR}}({{.IP_LOCAL_OUT_SOCK}}):u32 lport=+{{.INET_SOCK_LPORT}}({{.IP_LOCAL_OUT_SOCK}}):u16 raddr=+{{.INET_SOCK_RADDR}}({{.IP_LOCAL_OUT_SOCK}}):u32 rport=+{{.INET_SOCK_RPORT}}({{.IP_LOCAL_OUT_SOCK}}):u16",
			Filter:    "(af=={{.AF_INET}} || af=={{.AF_INET6}})",
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(ipLocalOutCall) }),
	},

	// Count received IPv4/TCP packets.
	//
	//  " tcp_v4_do_rcv(sock=0xffff9f1ddd216040) "
	{
		Probe: tracing.Probe{
			Name:      "tcp_v4_do_rcv_call",
			Address:   "tcp_v4_do_rcv",
			Fetchargs: "sock={{.P1}} size=+{{.SK_BUFF_LEN}}({{.P2}}):u32 laddr=+{{.INET_SOCK_LADDR}}({{.P1}}):u32 lport=+{{.INET_SOCK_LPORT}}({{.P1}}):u16 raddr=+{{.INET_SOCK_RADDR}}({{.P1}}):u32 rport=+{{.INET_SOCK_RPORT}}({{.P1}}):u16",
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(tcpV4DoRcv) }),
	},

	/***************************************************************************
	 * IPv4 / UDP
	 **************************************************************************/

	/* UDP/IPv4 send datagram. Good for counting payload bytes.
	   Also this should always be a packet. If we find a way to count packets
	   Here and ignore ip_local_out for UDP, it might avoid large-offload issues.
	*/
	{
		Probe: tracing.Probe{
			Name:      "udp_sendmsg_in",
			Address:   "udp_sendmsg",
			Fetchargs: "sock={{.UDP_SENDMSG_SOCK}} size={{.UDP_SENDMSG_LEN}} laddr=+{{.INET_SOCK_LADDR}}({{.UDP_SENDMSG_SOCK}}):u32 lport=+{{.INET_SOCK_LPORT}}({{.UDP_SENDMSG_SOCK}}):u16 raddr=+{{.SOCKADDR_IN_ADDR}}(+0({{.UDP_SENDMSG_MSG}})):u32 siptr=+0({{.UDP_SENDMSG_MSG}}) siaf=+{{.SOCKADDR_IN_AF}}(+0({{.UDP_SENDMSG_MSG}})):u16 rport=+{{.SOCKADDR_IN_PORT}}(+0({{.UDP_SENDMSG_MSG}})):u16 altraddr=+{{.INET_SOCK_RADDR}}({{.UDP_SENDMSG_SOCK}}):u32 altrport=+{{.INET_SOCK_RPORT}}({{.UDP_SENDMSG_SOCK}}):u16",
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(udpSendMsgCall) }),
	},

	{
		Probe: tracing.Probe{
			Name:      "udp_queue_rcv_skb",
			Address:   "udp_queue_rcv_skb",
			Fetchargs: "sock={{.P1}} size=+{{.SK_BUFF_LEN}}({{.P2}}):u32 laddr=+{{.INET_SOCK_LADDR}}({{.P1}}):u32 lport=+{{.INET_SOCK_LPORT}}({{.P1}}):u16 iphdr=+{{.SK_BUFF_NETWORK}}({{.P2}}):u16 udphdr=+{{.SK_BUFF_TRANSPORT}}({{.P2}}):u16 base=+{{.SK_BUFF_HEAD}}({{.P2}}) packet=" + helper.MakeMemoryDump("+{{.SK_BUFF_HEAD}}({{.P2}})", 0, skBuffDataDumpBytes),
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(udpQueueRcvSkb) }),
	},

	/***************************************************************************
	 * Clock Sync
	 **************************************************************************/

	/* This probe is used as a clock synchronization signal
	 */
	{
		Probe: tracing.Probe{
			Name:      "clock_sync_probe",
			Address:   "{{.SYS_UNAME}}",
			Fetchargs: "magic=+0({{.SYS_P1}}):u64 timestamp=+8({{.SYS_P1}}):u64",
			Filter:    fmt.Sprintf("magic==0x%x", clockSyncMagic),
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(clockSyncCall) }),
	},
}

// KProbes used only when IPv6 is disabled.
var ipv4OnlyKProbes = []helper.ProbeDef{
	// Return of accept(). Local side is usually zero so not fetched. Needs
	// further I/O to populate source.Good for marking a connection as inbound.
	//
	//  " <- accept(sock=0xffff9f1ddc5eb780, raddr=10.0.2.15, rport=22) "
	{
		Probe: tracing.Probe{
			Type:    tracing.TypeKRetProbe,
			Name:    "inet_csk_accept_ret4",
			Address: "inet_csk_accept",
			Fetchargs: "sock={{.RET}} laddr=+{{.INET_SOCK_LADDR}}({{.RET}}):u32 lport=+{{.INET_SOCK_LPORT}}({{.RET}}):u16 raddr=+{{.INET_SOCK_RADDR}}({{.RET}}):u32 rport=+{{.INET_SOCK_RPORT}}({{.RET}}):u16 " +
				"family=+{{.INET_SOCK_AF}}({{.RET}}):u16",
			Filter: "family=={{.AF_INET}}",
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(tcpAcceptResult4) }),
	},

	// Data is sent via TCP.
	// Good for (payload) data counters and getting full sock src and dest.
	// Not valid for packet counters, sock behaves as a stream.
	//
	//  " tcp_sendmsg(sock=0xffff9f1ddd216040, len=517, 10.0.2.15:55310 -> 151.101.66.217:443) "
	{
		Probe: tracing.Probe{
			Name:    "tcp_sendmsg_in4",
			Address: "tcp_sendmsg",
			Fetchargs: "sock={{.TCP_SENDMSG_SOCK}} size={{.TCP_SENDMSG_LEN}} laddr=+{{.INET_SOCK_LADDR}}({{.TCP_SENDMSG_SOCK}}):u32 lport=+{{.INET_SOCK_LPORT}}({{.TCP_SENDMSG_SOCK}}):u16 raddr=+{{.INET_SOCK_RADDR}}({{.TCP_SENDMSG_SOCK}}):u32 rport=+{{.INET_SOCK_RPORT}}({{.TCP_SENDMSG_SOCK}}):u16 " +
				"family=+{{.INET_SOCK_AF}}({{.TCP_SENDMSG_SOCK}}):u16",
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(tcpSendMsgCall4) }),
	},
}

// KProbes used when IPv6 is enabled.
var ipv6KProbes = []helper.ProbeDef{
	//***************************************************************************
	//* IPv6
	//***************************************************************************

	// IPv6 socket created. Good for associating sockets with pids.
	// ** This is a struct socket* not a struct sock* **
	//
	// inet6_create() is handled the same as inet_create()
	//
	//  " inet_create(socket=0xffff9f1ddadb8080, proto=17) "
	{
		Probe: tracing.Probe{
			Name:      "inet6_create",
			Address:   "inet6_create",
			Fetchargs: "proto={{.P3}}:s32",
			// proto=0 will select the protocol by looking at socket type (STREAM|DGRAM)
			Filter: "proto==0 || proto=={{.IPPROTO_TCP}} || proto=={{.IPPROTO_UDP}}",
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(inetCreate) }),
	},

	/***************************************************************************
	 * IPv6/TCP
	 **************************************************************************/

	// IPv6 TCP packet is sent. Acceptable as a packet counter,
	// But the actual data sent might span multiple packets if TSO is in use.
	//
	// (lport is fetched just for the sake of dev mode filtering).
	//
	// This call is asumed to have the same arguments as ip_local_out (ipv4)
	//
	//  " inet6_csk_xmit(sock=0xffff9f1ddd216040) "
	{
		Probe: tracing.Probe{
			Name:      "inet6_csk_xmit_call",
			Address:   "inet6_csk_xmit",
			Fetchargs: "sock={{.INET6_CSK_XMIT_SOCK}} size=+{{.SK_BUFF_LEN}}({{.INET6_CSK_XMIT_SKBUFF}}):u32 lport=+{{.INET_SOCK_LPORT}}({{.INET6_CSK_XMIT_SOCK}}):u16 rport=+{{.INET_SOCK_RPORT}}({{.INET6_CSK_XMIT_SOCK}}):u16 laddr6a={{.INET_SOCK_V6_LADDR_A}}({{.INET6_CSK_XMIT_SOCK}}){{.INET_SOCK_V6_TERM}} laddr6b={{.INET_SOCK_V6_LADDR_B}}({{.INET6_CSK_XMIT_SOCK}}){{.INET_SOCK_V6_TERM}} raddr6a={{.INET_SOCK_V6_RADDR_A}}({{.INET6_CSK_XMIT_SOCK}}){{.INET_SOCK_V6_TERM}} raddr6b={{.INET_SOCK_V6_RADDR_B}}({{.INET6_CSK_XMIT_SOCK}}){{.INET_SOCK_V6_TERM}}",
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(inet6CskXmitCall) }),
	},

	// Count received IPv6/TCP packets.
	//
	//  " tcp_v6_do_rcv(sock=0xffff9f1ddd216040) "
	{
		Probe: tracing.Probe{
			Name:      "tcp_v6_do_rcv_call",
			Address:   "tcp_v6_do_rcv",
			Fetchargs: "sock={{.P1}} size=+{{.SK_BUFF_LEN}}({{.P2}}):u32 lport=+{{.INET_SOCK_LPORT}}({{.P1}}):u16 rport=+{{.INET_SOCK_RPORT}}({{.P1}}):u16 laddr6a={{.INET_SOCK_V6_LADDR_A}}({{.P1}}){{.INET_SOCK_V6_TERM}} laddr6b={{.INET_SOCK_V6_LADDR_B}}({{.P1}}){{.INET_SOCK_V6_TERM}} raddr6a={{.INET_SOCK_V6_RADDR_A}}({{.P1}}){{.INET_SOCK_V6_TERM}} raddr6b={{.INET_SOCK_V6_RADDR_B}}({{.P1}}){{.INET_SOCK_V6_TERM}}",
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(tcpV6DoRcv) }),
	},

	// An IPv6 / TCP socket connect attempt:
	//
	//  " connect6(sock=0xffff9f1ddd216040, 0.0.0.0:0 -> 151.101.66.217:443) "
	{
		Probe: tracing.Probe{
			Name:      "tcp6_connect_in",
			Address:   "tcp_v6_connect",
			Fetchargs: "sock={{.P1}} laddra={{.INET_SOCK_V6_LADDR_A}}({{.P1}}){{.INET_SOCK_V6_TERM}} laddrb={{.INET_SOCK_V6_LADDR_B}}({{.P1}}){{.INET_SOCK_V6_TERM}} lport=+{{.INET_SOCK_LPORT}}({{.P1}}):u16 af=+{{.SOCKADDR_IN6_AF}}({{.P2}}):u16 addra=+{{.SOCKADDR_IN6_ADDRA}}({{.P2}}):u64 addrb=+{{.SOCKADDR_IN6_ADDRB}}({{.P2}}):u64 port=+{{.SOCKADDR_IN6_PORT}}({{.P2}}):u16",
			Filter:    "af=={{.AF_INET6}}",
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(tcpIPv6ConnectCall) }),
	},

	// Result of IPv6/TCP connect:
	//
	//  " <- connect ok (retval==0 or retval==-ERRNO) "
	{
		Probe: tracing.Probe{
			Type:      tracing.TypeKRetProbe,
			Name:      "tcp6_connect_out",
			Address:   "tcp_v6_connect",
			Fetchargs: "retval={{.RET}}:s32",
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(tcpConnectResult) }),
	},

	/***************************************************************************
	 * IPv6/UDP
	 **************************************************************************/

	/* UDP/IPv6 send datagram. Good for counting payload bytes.
	   Also this should always be a packet. If we find a way to count packets
	   Here and ignore ip_local_out for UDP, it might avoid large-offload issues.

	   Has the same argument format as udp_sendmsg as both were updated (1st arg
	   removed) in the same commit to the kernel.
	*/
	{
		Probe: tracing.Probe{
			Name:      "udpv6_sendmsg_in",
			Address:   "udpv6_sendmsg",
			Fetchargs: "sock={{.UDP_SENDMSG_SOCK}} size={{.UDP_SENDMSG_LEN}} laddra={{.INET_SOCK_V6_LADDR_A}}({{.UDP_SENDMSG_SOCK}}){{.INET_SOCK_V6_TERM}} laddrb={{.INET_SOCK_V6_LADDR_B}}({{.UDP_SENDMSG_SOCK}}){{.INET_SOCK_V6_TERM}} lport=+{{.INET_SOCK_LPORT}}({{.UDP_SENDMSG_SOCK}}):u16 raddra=+{{.SOCKADDR_IN6_ADDRA}}(+0({{.UDP_SENDMSG_MSG}})):u64 raddrb=+{{.SOCKADDR_IN6_ADDRB}}(+0({{.UDP_SENDMSG_MSG}})):u64 rport=+{{.SOCKADDR_IN6_PORT}}(+0({{.UDP_SENDMSG_MSG}})):u16 altraddra={{.INET_SOCK_V6_RADDR_A}}({{.UDP_SENDMSG_SOCK}}){{.INET_SOCK_V6_TERM}} altraddrb={{.INET_SOCK_V6_RADDR_B}}({{.UDP_SENDMSG_SOCK}}){{.INET_SOCK_V6_TERM}} altrport=+{{.INET_SOCK_RPORT}}({{.UDP_SENDMSG_SOCK}}):u16 si6ptr=+0({{.UDP_SENDMSG_MSG}}) si6af=+{{.SOCKADDR_IN6_AF}}(+0({{.UDP_SENDMSG_MSG}})):u16",
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(udpv6SendMsgCall) }),
	},

	/* UDP/IPv6 receive datagram. Good for counting payload bytes and packets.*/
	{
		Probe: tracing.Probe{
			Name:      "udpv6_queue_rcv_skb",
			Address:   "udpv6_queue_rcv_skb",
			Fetchargs: "sock={{.P1}} size=+{{.SK_BUFF_LEN}}({{.P2}}):u32 laddra={{.INET_SOCK_V6_LADDR_A}}({{.P1}}){{.INET_SOCK_V6_TERM}} laddrb={{.INET_SOCK_V6_LADDR_B}}({{.P1}}){{.INET_SOCK_V6_TERM}} lport=+{{.INET_SOCK_LPORT}}({{.P1}}):u16 iphdr=+{{.SK_BUFF_NETWORK}}({{.P2}}):u16 udphdr=+{{.SK_BUFF_TRANSPORT}}({{.P2}}):u16 base=+{{.SK_BUFF_HEAD}}({{.P2}}) packet=" + helper.MakeMemoryDump("+{{.SK_BUFF_HEAD}}({{.P2}})", 0, skBuffDataDumpBytes),
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(udpv6QueueRcvSkb) }),
	},

	/***************************************************************************
	 * Dual IPv4 / IPv6 calls
	 **************************************************************************/

	// Data is sent via TCP (IPv4 or IPv6).
	// Good for (payload) data counters and getting full sock src and dest.
	// Not valid for packet counters, sock behaves as a stream.
	//
	//  " tcp_sendmsg(sock=0xffff9f1ddd216040, len=517, 10.0.2.15:55310 -> 151.101.66.217:443) "
	{
		Probe: tracing.Probe{
			Name:    "tcp_sendmsg_in",
			Address: "tcp_sendmsg",
			Fetchargs: "sock={{.TCP_SENDMSG_SOCK}} size={{.TCP_SENDMSG_LEN}} laddr=+{{.INET_SOCK_LADDR}}({{.TCP_SENDMSG_SOCK}}):u32 lport=+{{.INET_SOCK_LPORT}}({{.TCP_SENDMSG_SOCK}}):u16 raddr=+{{.INET_SOCK_RADDR}}({{.TCP_SENDMSG_SOCK}}):u32 rport=+{{.INET_SOCK_RPORT}}({{.TCP_SENDMSG_SOCK}}):u16 " +
				"family=+{{.INET_SOCK_AF}}({{.TCP_SENDMSG_SOCK}}):u16 laddr6a={{.INET_SOCK_V6_LADDR_A}}({{.TCP_SENDMSG_SOCK}}){{.INET_SOCK_V6_TERM}} laddr6b={{.INET_SOCK_V6_LADDR_B}}({{.TCP_SENDMSG_SOCK}}){{.INET_SOCK_V6_TERM}} raddr6a={{.INET_SOCK_V6_RADDR_A}}({{.TCP_SENDMSG_SOCK}}){{.INET_SOCK_V6_TERM}} raddr6b={{.INET_SOCK_V6_RADDR_B}}({{.TCP_SENDMSG_SOCK}}){{.INET_SOCK_V6_TERM}}",
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(tcpSendMsgCall) }),
	},

	// Return of accept(). Local side is usually zero so not fetched. Needs
	// further I/O to populate source.Good for marking a connection as inbound.
	//
	//  " <- accept(sock=0xffff9f1ddc5eb780, raddr=10.0.2.15, rport=22) "
	{
		Probe: tracing.Probe{
			Type:    tracing.TypeKRetProbe,
			Name:    "inet_csk_accept_ret",
			Address: "inet_csk_accept",
			Fetchargs: "sock={{.RET}} laddr=+{{.INET_SOCK_LADDR}}({{.RET}}):u32 lport=+{{.INET_SOCK_LPORT}}({{.RET}}):u16 raddr=+{{.INET_SOCK_RADDR}}({{.RET}}):u32 rport=+{{.INET_SOCK_RPORT}}({{.RET}}):u16 " +
				"family=+{{.INET_SOCK_AF}}({{.RET}}):u16 laddr6a={{.INET_SOCK_V6_LADDR_A}}({{.RET}}){{.INET_SOCK_V6_TERM}} laddr6b={{.INET_SOCK_V6_LADDR_B}}({{.RET}}){{.INET_SOCK_V6_TERM}} raddr6a={{.INET_SOCK_V6_RADDR_A}}({{.RET}}){{.INET_SOCK_V6_TERM}} raddr6b={{.INET_SOCK_V6_RADDR_B}}({{.RET}}){{.INET_SOCK_V6_TERM}}",
			Filter: "family=={{.AF_INET}} || family=={{.AF_INET6}}",
		},
		Decoder: helper.NewStructDecoder(func() interface{} { return new(tcpAcceptResult) }),
	},
}

func getKProbes(hasIPv6 bool) (list []helper.ProbeDef) {
	list = append(list, sharedKProbes...)
	if hasIPv6 {
		list = append(list, ipv6KProbes...)
	} else {
		list = append(list, ipv4OnlyKProbes...)
	}
	return list
}

func getAllKProbes() (list []helper.ProbeDef) {
	list = append(list, sharedKProbes...)
	list = append(list, ipv6KProbes...)
	list = append(list, ipv4OnlyKProbes...)
	return list
}
