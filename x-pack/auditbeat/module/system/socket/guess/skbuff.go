// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build (linux && 386) || (linux && amd64)
// +build linux,386 linux,amd64

package guess

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/helper"
	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
)

/*
	Guess the offset of (struct sk_buff*)->len.

	This is tricky as an sk_buff usually has more memory allocated than its
	necessary to hold the payload, to make room for protocol headers.

	It analyses multiple sk_buff dumps and gets all the offsets that contain
	the payload size plus a constant between 0 and 128. Then it keeps the
	offset that consistently held size+C for the smallest possible
	constant C.

	Example iterations:
		iteration 1: {"HEADER_SIZES":[0,52],"OFF_0":[64],"OFF_52":[128]}
		iteration 2: {"HEADER_SIZES":[0,52],"OFF_0":[64],"OFF_52":[128]}
		iteration 3: {"HEADER_SIZES":[0,4,52,92],"OFF_0":[64],"OFF_4":[712],"OFF_52":[128],"OFF_92":[672]}
		iteration 4: {"HEADER_SIZES":[0,52],"OFF_0":[64],"OFF_52":[128]}

	Result:
	Guess guess_sk_buff_len completed: {"DETECTED_HEADER_SIZE":52,"SK_BUFF_LEN":128}
*/

const maxSafePayload = 508

func init() {
	if err := Registry.AddGuess(func() Guesser { return &guessSkBuffLen{} }); err != nil {
		panic(err)
	}
	if err := Registry.AddGuess(func() Guesser { return &guessSkBuffProto{} }); err != nil {
		panic(err)
	}
	if err := Registry.AddGuess(func() Guesser { return &guessSkBuffDataPtr{} }); err != nil {
		panic(err)
	}
}

type guessSkBuffLen struct {
	ctx     Context
	cs      inetClientServer
	written int
}

// Name of this guess.
func (g *guessSkBuffLen) Name() string {
	return "guess_sk_buff_len"
}

// Provides returns the list of variables discovered.
func (g *guessSkBuffLen) Provides() []string {
	return []string{
		"SK_BUFF_LEN",
		"DETECTED_HEADER_SIZE",
	}
}

// Requires declares the variables required to run this guess.
func (g *guessSkBuffLen) Requires() []string {
	return []string{
		"IP_LOCAL_OUT",
		"IP_LOCAL_OUT_SK_BUFF",
	}
}

// Probes returns a probe on ip_local_out, which is called to output an IPv4
// packet.
func (g *guessSkBuffLen) Probes() ([]helper.ProbeDef, error) {
	return []helper.ProbeDef{
		{
			Probe: tracing.Probe{
				Name:      "ip_local_out_len_guess",
				Address:   "{{.IP_LOCAL_OUT}}",
				Fetchargs: helper.MakeMemoryDump("{{.IP_LOCAL_OUT_SK_BUFF}}", 0, skbuffDumpSize),
			},
			Decoder: tracing.NewDumpDecoder,
		},
	}, nil
}

// Prepare creates a connected TCP client-server.
func (g *guessSkBuffLen) Prepare(ctx Context) error {
	g.ctx = ctx
	return g.cs.SetupTCP()
}

// Terminate cleans up the server.
func (g *guessSkBuffLen) Terminate() error {
	return g.cs.Cleanup()
}

// Trigger causes a packet with a random payload size to be output.
func (g *guessSkBuffLen) Trigger() error {
	const minPayload = 213
	n := minPayload + rand.Intn(maxSafePayload+1-minPayload)
	buf := make([]byte, n)
	var err error
	g.written, err = unix.SendmsgN(g.cs.client, buf, nil, nil, 0)
	if err != nil {
		return err
	}
	unix.Read(g.cs.accepted, buf)
	return nil
}

// Extract scans the sk_buff memory for any values between the expected
// payload + [0 ... 128).
func (g *guessSkBuffLen) Extract(ev interface{}) (common.MapStr, bool) {
	skbuff := ev.([]byte)
	if len(skbuff) != skbuffDumpSize || g.written <= 0 {
		return nil, false
	}
	const (
		uIntSize          = 4
		n                 = skbuffDumpSize / uIntSize
		maxOverhead       = 128
		minHeadersSize    = 0 // 20 /* min IP*/ + 20 /* min TCP */
		ipHeaderSizeChunk = 4
	)
	target := uint32(g.written)
	arr := (*[n]uint32)(unsafe.Pointer(&skbuff[0]))[:]
	var results [maxOverhead][]int
	for i := 0; i < n; i++ {
		if val := arr[i]; val >= target && val < target+maxOverhead {
			excess := val - target
			results[excess] = append(results[excess], i*uIntSize)
		}
	}

	result := make(common.MapStr)
	var overhead []int
	for i := minHeadersSize; i < maxOverhead; i += ipHeaderSizeChunk {
		if len(results[i]) > 0 {
			result[fmt.Sprintf("OFF_%d", i)] = results[i]
			overhead = append(overhead, i)
		}
	}
	if len(overhead) == 0 {
		return nil, false
	}
	result["HEADER_SIZES"] = overhead
	return result, true
}

// NumRepeats configures this guess to be repeated 4 times.
func (g *guessSkBuffLen) NumRepeats() int {
	return 4
}

// Reduce takes the output from the multiple runs and returns the offset
// which consistently returned the expected length plus a fixed constant.
func (g *guessSkBuffLen) Reduce(results []common.MapStr) (result common.MapStr, err error) {
	clones := make([]common.MapStr, 0, len(results))
	for _, res := range results {
		val, found := res["HEADER_SIZES"]
		if !found {
			return nil, errors.New("not all attempts detected offsets")
		}
		m := make(common.MapStr, 1)
		m["HEADER_SIZES"] = val
		clones = append(clones, m)
	}
	if result, err = consolidate(clones); err != nil {
		return nil, err
	}

	list, err := getListField(result, "HEADER_SIZES")
	if err != nil {
		return nil, err
	}
	headerSize := list[0]
	if len(list) > 1 && headerSize == 0 {
		// There's two lengths in the sk_buff, one is the payload length
		// the other one is payload + headers.
		// Keep the second as we want to count the whole packet size.
		headerSize = list[1]
	}
	key := fmt.Sprintf("OFF_%d", headerSize)
	for idx, m := range clones {
		delete(m, "HEADER_SIZES")
		m[key] = results[idx][key]
	}

	if result, err = consolidate(clones); err != nil {
		return nil, err
	}
	list, err = getListField(result, key)
	if err != nil {
		return nil, err
	}

	return common.MapStr{
		"SK_BUFF_LEN":          list[0],
		"DETECTED_HEADER_SIZE": headerSize,
	}, nil
}

/* guess_sk_buff_proto
   This guesses the offset of the sk_buff->protocol field. This field holds
   the frame format / ethernet protocol.

   Each run it sends an UDP datagram between two sockets, alternating between
   IPv4 and IPv6. Then scans the sk_buff returned by __skb_recv_datagram
   for the values:

   ETH_P_IP   0x0800
   ETH_P_IPV6 0x86dd

   Returns:

	SK_BUFF_PROTO: 128
*/

type guessSkBuffProto struct {
	ctx                    Context
	doIPv6                 bool
	hasIPv6                bool
	cs                     inetClientServer
	loopback               helper.IPv6Loopback
	clientAddr, serverAddr unix.SockaddrInet6
	client, server         int
	msg                    []byte
}

// Name is the name of this probe.
func (g *guessSkBuffProto) Name() string {
	return "guess_sk_buff_proto"
}

// Probes returns a kretprobe in __skb_recv_datagram that dumps the memory
// pointed to by the return value (a struct sk_buff*).
func (g *guessSkBuffProto) Probes() ([]helper.ProbeDef, error) {
	return []helper.ProbeDef{
		{
			Probe: tracing.Probe{
				Type:      tracing.TypeKRetProbe,
				Name:      "guess_recv_datagram",
				Address:   "{{.RECV_UDP_DATAGRAM}}",
				Fetchargs: helper.MakeMemoryDump("{{.RET}}", 0, 1024),
			},
			Decoder: tracing.NewDumpDecoder,
		},
	}, nil
}

// Provides returns the list of variables provided by this probe.
func (g *guessSkBuffProto) Provides() []string {
	return []string{
		"SK_BUFF_PROTO",
	}
}

// Requires returns the list of variables required by this probe.
func (g *guessSkBuffProto) Requires() []string {
	return nil
}

// Prepare sets up either two UDP sockets, using either IPv4 or IPv6.
func (g *guessSkBuffProto) Prepare(ctx Context) (err error) {
	g.ctx = ctx
	g.hasIPv6, err = isIPv6Enabled(ctx.Vars)
	if err != nil {
		return fmt.Errorf("unable to determine if IPv6 is enabled: %w", err)
	}
	g.doIPv6 = g.hasIPv6 && !g.doIPv6
	g.msg = make([]byte, 0x123)
	if g.doIPv6 {
		g.loopback, err = helper.NewIPv6Loopback()
		if err != nil {
			return fmt.Errorf("detect IPv6 loopback failed: %w", err)
		}
		defer func() {
			if err != nil {
				g.loopback.Cleanup()
			}
		}()
		clientIP, err := g.loopback.AddRandomAddress()
		if err != nil {
			return fmt.Errorf("failed adding first device address: %w", err)
		}
		serverIP, err := g.loopback.AddRandomAddress()
		if err != nil {
			return fmt.Errorf("failed adding second device address: %w", err)
		}
		copy(g.clientAddr.Addr[:], clientIP)
		copy(g.serverAddr.Addr[:], serverIP)

		if g.client, g.clientAddr, err = createSocket6WithProto(unix.SOCK_DGRAM, g.clientAddr); err != nil {
			return fmt.Errorf("error creating server: %w", err)
		}
		if g.server, g.serverAddr, err = createSocket6WithProto(unix.SOCK_DGRAM, g.serverAddr); err != nil {
			return fmt.Errorf("error creating client: %w", err)
		}
	} else {
		g.cs.SetupUDP()
	}
	return nil
}

// Terminate cleans up the sockets.
func (g *guessSkBuffProto) Terminate() (err error) {
	if g.doIPv6 {
		unix.Close(g.client)
		unix.Close(g.server)
		err = g.loopback.Cleanup()
	} else {
		err = g.cs.Cleanup()
	}
	return
}

// Trigger sends a packet from the client and receives it in the server socket.
func (g *guessSkBuffProto) Trigger() error {
	if g.doIPv6 {
		if err := unix.Sendto(g.client, g.msg, 0, &g.serverAddr); err != nil {
			return fmt.Errorf("failed to send ipv4: %w", err)
		}
		if _, _, err := unix.Recvfrom(g.server, g.msg, 0); err != nil {
			return fmt.Errorf("failed to receive ipv4: %w", err)
		}
	} else {
		if err := unix.Sendto(g.cs.client, g.msg, 0, &g.cs.srvAddr); err != nil {
			return fmt.Errorf("failed to send ipv4: %w", err)
		}
		if _, _, err := unix.Recvfrom(g.cs.server, g.msg, 0); err != nil {
			return fmt.Errorf("failed to receive ipv4: %w", err)
		}
	}
	return nil
}

// Extract will scan the sk_buff memory to look for all the uint16-sized memory
// locations that contain the expected protocol value.
func (g *guessSkBuffProto) Extract(event interface{}) (common.MapStr, bool) {
	raw := event.([]byte)
	needle := []byte{0x08, 0x00} // ETH_P_IP
	if g.doIPv6 {
		needle = []byte{0x86, 0xdd} // ETH_P_IPV6
	}
	var hits []int
	off := indexAligned(raw, needle, 0, 2)
	for off != -1 {
		hits = append(hits, off)
		off = indexAligned(raw, needle, off+2, 2)
	}

	return common.MapStr{
		"SK_BUFF_PROTO": hits,
	}, true
}

// NumRepeats returns the number of repeats for this probe.
func (g *guessSkBuffProto) NumRepeats() int {
	return 8
}

// Reduce uses the partial results from every repetition to figure out the
// right offset of the protocol field.
func (g *guessSkBuffProto) Reduce(results []common.MapStr) (result common.MapStr, err error) {
	if result, err = consolidate(results); err != nil {
		return nil, err
	}

	for _, key := range []string{"SK_BUFF_PROTO"} {
		list, err := getListField(result, key)
		if err != nil {
			return nil, err
		}
		result[key] = list[0]
	}
	return result, nil
}

const (
	payloadLen    = 0x123
	dataDumpBytes = 256
)

/*
	guess_sk_buff_data_ptr

	This guesses the offsets of the following fields within an sk_buff:

		sk_buff_data_t transport_header;
		sk_buff_data_t network_header;
		sk_buff_data_t mac_header;
		[...]
		unsigned char *head;

	These fields all appear after protocol (SK_BUFF_PROTO) discovered above.

	sk->head is the pointer to the packet data contained in the sk_buff.
	sk->x_header is the offset(or pointer, see below) of the transport/net/mac
	header within the packet data.

	This fields have changed a few times over the years. Before kernel 3.11
	they had type "sk_buff_data_t", which in turn was defined in a conditional
	macro.
		(pre-3.11, 32 bits) sk_buff_data_t is a pointer.
		(pre-3.11, 64 bits) sk_buff_data_t is an unsigned int index.
		(post-3.11)         sk_buff_data_t replaced with uint16.

	HOW THIS GUESS WORKS

	This guess executes an undefined number of times. While this is capped at
    64 tries, it succeeds after 2 to 4 repeats (depending of the kernel).

	First it discovers the offset of the "head" pointer. This is done by
    sequentially treating as a pointer all the pointer-aligned memory addresses
	after "protocol". For every candidate pointer, it dumps the memory pointed
	by it (if any) and the pointer value itself. Another probe set to the same
	function dumps the sk_buff.

	Extract receives the candidate dump and the sk_buff.

	First it will check if the dumped data is the packet. For this it takes
    advantage of knowing the src/dst ip and ports. This helps discover the offset
	of the IP and UDP headers within the packet.

	Then it scans the sk_buff itself, using the learned information to discover
	the offsets/pointers for the protocol headers.

	If at any step it fails to find the expected data, it bails out and repeats
	the guess, using the next pointer in the sk_buff.

	----

	This is how the "head" region looks for an UDP packet and
	addresses 7f12a0d7 7f1033bd:

	The IP header is at 0x10, the MAC header starts at 0x02 (those extra 2 bytes
	seem to be there to align the IPs to a 4-byte boundary and because a mac
	header offset of 0 is treated as invalid). The UDP header is at 0x24.

	00000000  10 34 00 00 00 00 00 00  00 00 00 00 00 00 08 00  |.4..............|
	00000010  45 00 01 3f 00 00 40 00  40 11 66 f7 7f 12 a0 d7  |E..?..@.@.f.....|
	00000020  7f 10 33 bd be a6 8f 29  01 2b d3 f3 48 45 4c 4c  |..3....).+..HELL|
	00000030  4f 21 48 45 4c 4c 4f 21  48 45 4c 4c 4f 21 48 45  |O!HELLO!HELLO!HE|
	00000040  4c 4c 4f 21 48 45 4c 4c  4f 21 48 45 4c 4c 4f 21  |LLO!HELLO!HELLO!|
	00000050  48 45 4c 4c 4f 21 48 45  4c 4c 4f 21 48 45 4c 4c  |HELLO!HELLO!HELL|
	00000060  4f 21 48 45 4c 4c 4f 21  48 45 4c 4c 4f 21 48 45  |O!HELLO!HELLO!HE|
	00000070  4c 4c 4f 21 48 45 4c 4c  4f 21 48 45 4c 4c 4f 21  |LLO!HELLO!HELLO!|
	00000080  48 45 4c 4c 4f 21 48 45  4c 4c 4f 21 48 45 4c 4c  |HELLO!HELLO!HELL|
	00000090  4f 21 48 45 4c 4c 4f 21  48 45 4c 4c 4f 21 48 45  |O!HELLO!HELLO!HE|
	000000a0  4c 4c 4f 21 48 45 4c 4c  4f 21 48 45 4c 4c 4f 21  |LLO!HELLO!HELLO!|
	000000b0  48 45 4c 4c 4f 21 48 45  4c 4c 4f 21 48 45 4c 4c  |HELLO!HELLO!HELL|
	000000c0  4f 21 48 45 4c 4c 4f 21  48 45 4c 4c 4f 21 48 45  |O!HELLO!HELLO!HE|
	000000d0  4c 4c 4f 21 48 45 4c 4c  4f 21 48 45 4c 4c 4f 21  |LLO!HELLO!HELLO!|
	000000e0  48 45 4c 4c 4f 21 48 45  4c 4c 4f 21 48 45 4c 4c  |HELLO!HELLO!HELL|
	000000f0  4f 21 48 45 4c 4c 4f 21  48 45 4c 4c 4f 21 48 45  |O!HELLO!HELLO!HE|

	This is how the matching sk_buff looks:

	The pointer to the sk_buff is 0xffff880037c43c00 at 0xd0.
	The offsets (in this case) for transport, network and mac headers are at
	0xbc, 0xc0 and 0xc4 (size uint32 in this case):

	00000000  00 00 00 00 00 00 00 00  00 00 00 00 00 00 00 00  |................|
	00000010  00 c4 ea 3d 00 88 ff ff  00 00 00 00 00 00 00 00  |...=............|
	00000020  00 00 00 00 00 00 00 00  c0 46 a1 3a 00 88 ff ff  |.........F.:....|
	00000030  00 00 00 00 00 00 00 00  00 00 00 00 00 00 00 00  |................|
	00000040  00 00 00 00 00 00 00 00  00 00 00 00 00 00 00 00  |................|
	00000050  2b 01 00 00 00 00 00 00  00 00 00 00 00 00 00 00  |+...............|
	00000060  00 00 00 00 00 00 00 00  2b 01 00 00 00 00 00 00  |........+.......|
	00000070  0e 00 00 00 24 00 06 00  00 00 00 00 0d 00 08 00  |....$...........|
	00000080  20 04 47 81 ff ff ff ff  00 00 00 00 00 00 00 00  | .G.............|
	00000090  00 00 00 00 00 00 00 00  00 00 00 00 00 00 00 00  |................|
	000000a0  01 00 00 00 00 00 00 00  00 00 00 00 00 00 00 00  |................|
	000000b0  00 00 00 00 00 00 00 00  00 00 00 00 24 00 00 00  |............$...|
	000000c0  10 00 00 00 02 00 00 00  4f 01 00 00 80 01 00 00  |........O.......|
	000000d0  00 3c c4 37 00 88 ff ff  24 3c c4 37 00 88 ff ff  |.<.7....$<.7....|
	000000e0  68 02 00 00 01 00 00 00  57 49 50 5f 49 4e 45 54  |h.......WIP_INET|
	000000f0  00 4c 4f 47 5f 47 52 4f  55 50 5f 44 45 56 5f 48  |.LOG_GROUP_DEV_H|
	00000100  00 00 00 00 00 00 00 00  00 00 00 00 00 00 00 00  |................|

*/

type guessSkBuffDataPtr struct {
	ctx Context
	cs  inetClientServer
	// offset of proto (already discovered)
	protoOffset int
	// dump offset of current iteration
	dumpOffset int
	payload    []byte
	// results of the 2 kprobes installed
	data   *dataDump
	skbuff []byte
}

// A dump of sk_buff->data
type dataDump struct {
	Ptr  uintptr             `kprobe:"ptr"`
	Data [dataDumpBytes]byte `kprobe:"data,greedy"`
}

// Name of this guess.
func (g *guessSkBuffDataPtr) Name() string {
	return "guess_sk_buff_data_ptr"
}

// Probes return two probes at the same function. One will output a *dataDump
// the other one a []byte with the sk_buff.
func (g *guessSkBuffDataPtr) Probes() ([]helper.ProbeDef, error) {
	address := fmt.Sprintf("+%d(%s)", g.dumpOffset, g.ctx.Vars["RET"])
	return []helper.ProbeDef{
		{
			Probe: tracing.Probe{
				Type:      tracing.TypeKRetProbe,
				Name:      "guess_recv_dgrm_data",
				Address:   "{{.RECV_UDP_DATAGRAM}}",
				Fetchargs: fmt.Sprintf("ptr=%s data=%s", address, helper.MakeMemoryDump(address, 0, dataDumpBytes)),
			},
			Decoder: helper.NewStructDecoder(func() interface{} { return new(dataDump) }),
		},
		{
			Probe: tracing.Probe{
				Type:      tracing.TypeKRetProbe,
				Name:      "guess_recv_dgrm_skbuff",
				Address:   "{{.RECV_UDP_DATAGRAM}}",
				Fetchargs: helper.MakeMemoryDump("{{.RET}}", 0, skbuffDumpSize),
			},
			Decoder: tracing.NewDumpDecoder,
		},
	}, nil
}

// Provides returns the list of variables discovered.
func (g *guessSkBuffDataPtr) Provides() []string {
	return []string{
		"SK_BUFF_HEAD",
	}
}

// Requires declares the variables required to run this guess.
func (g *guessSkBuffDataPtr) Requires() []string {
	return []string{
		"SK_BUFF_PROTO",
	}
}

// Prepare initializes the guess. The first time it sets protoOffset/dumpOffset
// and payload. When called again it increments the dumpOffset.
func (g *guessSkBuffDataPtr) Prepare(ctx Context) (err error) {
	g.ctx = ctx
	// start looking for the pointer at the next uintptr aligned offset after
	// protocol field.
	if g.dumpOffset == 0 {
		g.protoOffset = ctx.Vars["SK_BUFF_PROTO"].(int)
		g.dumpOffset = alignTo(g.protoOffset+2, int(sizeOfPtr))
		g.payload = make([]byte, payloadLen)
		banner := "HELLO!"
		for i := 0; i < payloadLen; i++ {
			g.payload[i] = banner[i%len(banner)]
		}
	} else {
		// keep looking at the next pointer
		g.dumpOffset += int(sizeOfPtr)
	}
	return g.cs.SetupUDP()
}

// Terminate resets the saved buffers and closes the UDP sockets.
func (g *guessSkBuffDataPtr) Terminate() error {
	g.data = nil
	g.skbuff = nil
	return g.cs.Cleanup()
}

// Trigger causes a packet to be received at server socket.
func (g *guessSkBuffDataPtr) Trigger() error {
	if err := unix.Sendto(g.cs.client, g.payload, 0, &g.cs.srvAddr); err != nil {
		return fmt.Errorf("failed to send ipv4: %w", err)
	}
	if _, _, err := unix.Recvfrom(g.cs.server, g.payload, 0); err != nil {
		return fmt.Errorf("failed to receive ipv4: %w", err)
	}
	return nil
}

func pointerAt(buf []byte) uintptr {
	if sizeOfPtr == 8 {
		return uintptr(tracing.MachineEndian.Uint64(buf[:8]))
	}
	return uintptr(tracing.MachineEndian.Uint32(buf[:4]))
}

func u16At(buf []byte) uintptr {
	return uintptr(tracing.MachineEndian.Uint16(buf[:2]))
}

func u32At(buf []byte) uintptr {
	return uintptr(tracing.MachineEndian.Uint32(buf[:4]))
}

// Extract stores the result of each kretprobe and when it received both
// it makes validations.
//
// As this is an EventualGuess, the return values of Extract are as follows:
// (any), false : Signals that it needs another event in the current iteration.
// nil, true : Finish the current iteration and perform a new one.
// (non-nil), true : The guess completed.
func (g *guessSkBuffDataPtr) Extract(event interface{}) (common.MapStr, bool) {
	switch v := event.(type) {
	case *dataDump:
		g.data = v
	case []byte:
		g.skbuff = v
	}
	if g.data == nil || g.skbuff == nil {
		// wait for the missing event
		return nil, false
	}

	if g.dumpOffset >= skbuffDumpSize {
		// Scanned more memory than its available from sk_buff. Bail out.
		// Returns a non-nil map so the guess is not repeated until MaxRepeats()
		return common.MapStr{"FAILED": true}, true
	}

	//
	// Received both events.
	//

	// Check that we got the right sk_buff:

	// See if g.data is a valid sk_buff->data (contains our packet).
	var ipAddresses [8]byte
	copy(ipAddresses[:4], g.cs.cliAddr.Addr[:]) // ipv4 source
	copy(ipAddresses[4:], g.cs.srvAddr.Addr[:]) // ipv4 dest
	var ports [4]byte
	binary.BigEndian.PutUint16(ports[:2], uint16(g.cs.cliAddr.Port))
	binary.BigEndian.PutUint16(ports[2:], uint16(g.cs.srvAddr.Port))
	// Checking with align=1 although it always seems aligned at 4 because
	// sk_buff->data is always padded with 2 bytes at the start.
	ipHdrOff := indexAligned(g.data.Data[:], ipAddresses[:], 12 /*offset in iphdr*/ +14 /*eth header*/, 1)
	if ipHdrOff == -1 {
		// This is not out packet. dumpOffset is not the right one.
		return nil, true
	}
	ipHdrOff -= 12
	udpHdrOff := indexAligned(g.data.Data[:], ports[:], ipHdrOff+8, 1)
	if udpHdrOff == -1 {
		// This really should not happen
		g.ctx.Log.Debugf("%s found ip header but no udp header", g.Name())
		return nil, true
	}
	if g.data.Data[ipHdrOff]&0xf0 != 0x40 { // not an IP header?
		g.ctx.Log.Debugf("%s found ip header is not valid", g.Name())
		return nil, true
	}

	if len(g.skbuff) < g.dumpOffset+int(sizeOfPtr) || pointerAt(g.skbuff[g.dumpOffset:]) != g.data.Ptr {
		g.ctx.Log.Debugf("%s pointer at 0x%x is not valid %d %d %x %x\n%s\n",
			g.Name(), g.dumpOffset, len(g.skbuff), g.dumpOffset+int(sizeOfPtr), pointerAt(g.skbuff[g.dumpOffset:]), g.data.Ptr, hex.Dump(g.skbuff))
		return nil, true
	}

	//
	// Now make sense of the sk_buff:
	//
	// The fields we're looking for have had many different implementations:
	// before 3.11:
	//  - 32 bits: they're pointers.
	//  - 64 bits: they're ints.
	// after 3.11:
	//  - they're always u16
	//

	limit := g.protoOffset + 2
	protocolValue := binary.BigEndian.Uint16(g.skbuff[g.protoOffset:])
	scanFields := func(width int, ptrBase uintptr, reader func([]byte) uintptr) common.MapStr {
		var off [3]uintptr
		for base := g.dumpOffset - width*3; base >= limit; base -= width {
			for i := 0; i < 3; i++ {
				off[i] = reader(g.skbuff[base+i*width:]) - ptrBase
			}
			if off[0] == uintptr(udpHdrOff) &&
				off[1] == uintptr(ipHdrOff) &&
				off[2] > 0 &&
				off[2] != -ptrBase &&
				off[2] <= uintptr(ipHdrOff-14) &&
				binary.BigEndian.Uint16(g.data.Data[off[2]+12:]) == protocolValue {
				return common.MapStr{
					"SK_BUFF_HEAD":         g.dumpOffset,
					"SK_BUFF_TRANSPORT":    base,
					"SK_BUFF_NETWORK":      base + width,
					"SK_BUFF_MAC":          base + 2*width,
					"SK_BUFF_HAS_POINTERS": ptrBase != 0,
				}
			}
		}
		return nil
	}

	// search for u16 fields:
	if r := scanFields(2, 0, u16At); r != nil {
		return r, true
	}
	// Search for u32 fields:
	if r := scanFields(4, 0, u32At); r != nil {
		return r, true
	}
	// Search for pointers
	if r := scanFields(int(sizeOfPtr), g.data.Ptr, pointerAt); r != nil {
		return r, true
	}
	return nil, true
}

// MaxRepeats sets a size enough so that all the possible pointers in a sk_buff
// are scanned. It will never never repeat this many times due to an additional
// check in Extract().
func (g *guessSkBuffDataPtr) MaxRepeats() int {
	return 128
}
