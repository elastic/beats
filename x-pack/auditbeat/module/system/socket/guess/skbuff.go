// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package guess

import (
	"fmt"
	"math/rand"
	"unsafe"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/x-pack/auditbeat/module/system/socket/helper"
	"github.com/elastic/beats/x-pack/auditbeat/tracing"
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
	if err := Registry.AddGuess(&guessSkBuffLen{}); err != nil {
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
	const minPayload = 13
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

// Validate scans the sk_buff memory for any values between the expected
// payload + (0 .. 128).
func (g *guessSkBuffLen) Validate(ev interface{}) (common.MapStr, bool) {
	skbuff := ev.([]byte)
	if len(skbuff) != skbuffDumpSize || g.written <= 0 {
		return nil, false
	}
	const (
		uIntSize          = 4
		n                 = skbuffDumpSize / uIntSize
		maxOverhead       = 128
		minHeadersSize    = 0 //20 /* min IP*/ + 20 /* min TCP */
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
