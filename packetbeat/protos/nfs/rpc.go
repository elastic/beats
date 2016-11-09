// Package rpc provides support for parsing RPC messages and reporting the
// results. This package supports the RPC v2 protocol as defined by RFC 5531
// (RFC 1831).

package nfs

import (
	"encoding/binary"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"fmt"

	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/tcp"
	"github.com/elastic/beats/packetbeat/publish"
)

var debugf = logp.MakeDebug("rpc")

const (
	RPCLastFrag = 0x80000000
	RPCSizeMask = 0x7fffffff
)

const (
	RPCCall  = 0
	RPCReply = 1
)

type RPCStream struct {
	tcpTuple *common.TCPTuple
	rawData  []byte
}

type rpcConnectionData struct {
	Streams [2]*RPCStream
}

type RPC struct {
	// Configuration data.
	Ports              []int
	callsSeen          *common.Cache
	transactionTimeout time.Duration

	results publish.Transactions // Channel where results are pushed.
}

func init() {
	protos.Register("nfs", New)
}

func New(
	testMode bool,
	results publish.Transactions,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &RPC{}
	config := defaultConfig
	if !testMode {
		if err := cfg.Unpack(&config); err != nil {
			logp.Warn("failed to read config")
			return nil, err
		}
	}

	if err := p.init(results, &config); err != nil {
		logp.Warn("failed to init")
		return nil, err
	}
	return p, nil
}

func (rpc *RPC) init(results publish.Transactions, config *rpcConfig) error {
	rpc.setFromConfig(config)
	rpc.results = results
	rpc.callsSeen = common.NewCacheWithRemovalListener(
		rpc.transactionTimeout,
		protos.DefaultTransactionHashSize,
		func(k common.Key, v common.Value) {
			nfs, ok := v.(*NFS)
			if !ok {
				logp.Err("Expired value is not a MapStr (%T).", v)
				return
			}
			rpc.handleExpiredPacket(nfs)
		})

	rpc.callsSeen.StartJanitor(rpc.transactionTimeout)
	return nil
}

func (rpc *RPC) setFromConfig(config *rpcConfig) error {
	rpc.Ports = config.Ports
	rpc.transactionTimeout = config.TransactionTimeout
	return nil
}

func (rpc *RPC) GetPorts() []int {
	return rpc.Ports
}

// Called when TCP payload data is available for parsing.
func (rpc *RPC) Parse(
	pkt *protos.Packet,
	tcptuple *common.TCPTuple,
	dir uint8,
	private protos.ProtocolData,
) protos.ProtocolData {

	defer logp.Recover("ParseRPC exception")

	conn := ensureRPCConnection(private)

	conn = rpc.handleRPCFragment(conn, pkt, tcptuple, dir)
	if conn == nil {
		return nil
	}
	return conn
}

// Called when the FIN flag is seen in the TCP stream.
func (rpc *RPC) ReceivedFin(tcptuple *common.TCPTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {

	defer logp.Recover("ReceivedFinRpc exception")

	// forced by TCP interface
	return private
}

// Called when a packets are missing from the tcp
// stream.
func (rpc *RPC) GapInStream(tcptuple *common.TCPTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {

	defer logp.Recover("GapInRpcStream exception")

	// forced by TCP interface
	return private, false
}

// ConnectionTimeout returns the per stream connection timeout.
// Return <=0 to set default tcp module transaction timeout.
func (rpc *RPC) ConnectionTimeout() time.Duration {
	// forced by TCP interface
	return rpc.transactionTimeout
}

func ensureRPCConnection(private protos.ProtocolData) *rpcConnectionData {
	conn := getRPCConnection(private)
	if conn == nil {
		conn = &rpcConnectionData{}
	}
	return conn
}

func getRPCConnection(private protos.ProtocolData) *rpcConnectionData {
	if private == nil {
		return nil
	}

	priv, ok := private.(*rpcConnectionData)
	if !ok {
		logp.Warn("rpc connection data type error")
		return nil
	}
	if priv == nil {
		logp.Warn("Unexpected: rpc connection data not set")
		return nil
	}

	return priv
}

// Parse function is used to process TCP payloads.
func (rpc *RPC) handleRPCFragment(
	conn *rpcConnectionData,
	pkt *protos.Packet,
	tcptuple *common.TCPTuple,
	dir uint8,
) *rpcConnectionData {

	st := conn.Streams[dir]
	if st == nil {
		st = newStream(pkt, tcptuple)
		conn.Streams[dir] = st
	} else {
		// concatenate bytes
		st.rawData = append(st.rawData, pkt.Payload...)
		if len(st.rawData) > tcp.TCPMaxDataInStream {
			debugf("Stream data too large, dropping TCP stream")
			conn.Streams[dir] = nil
			return conn
		}
	}

	for len(st.rawData) > 0 {

		if len(st.rawData) < 4 {
			debugf("Wainting for more data")
			break
		}

		marker := uint32(binary.BigEndian.Uint32(st.rawData[0:4]))
		size := int(marker & RPCSizeMask)
		islast := (marker & RPCLastFrag) != 0

		if len(st.rawData)-4 < size {
			debugf("Wainting for more data")
			break
		}

		if !islast {
			logp.Warn("multifragment rpc message")
			break
		}

		xdr := &Xdr{data: st.rawData[4 : 4+size], offset: 0}

		// keep the rest of the next fragment
		st.rawData = st.rawData[4+size:]

		rpc.handleRPCPacket(xdr, pkt.Ts, tcptuple, dir)
	}

	return conn
}

func (rpc *RPC) handleRPCPacket(xdr *Xdr, ts time.Time, tcptuple *common.TCPTuple, dir uint8) {

	xid := fmt.Sprintf("%.8x", xdr.getUInt())

	msgType := xdr.getUInt()

	switch msgType {
	case RPCCall:
		rpc.handleCall(xid, xdr, ts, tcptuple, dir)
	case RPCReply:
		rpc.handleReply(xid, xdr, ts, tcptuple, dir)
	default:
		logp.Warn("Bad RPC message")
	}
}

func newStream(pkt *protos.Packet, tcptuple *common.TCPTuple) *RPCStream {
	return &RPCStream{
		tcpTuple: tcptuple,
		rawData:  pkt.Payload,
	}
}
