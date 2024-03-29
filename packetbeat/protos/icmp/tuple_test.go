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

//go:build !integration

package icmp

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIcmpTupleReverse(t *testing.T) {
	tuple := icmpTuple{
		icmpVersion: 4,
		srcIP:       net.IPv4(192, 168, 0, 1),
		dstIP:       net.IPv4(192, 168, 0, 2),
		id:          256,
		seq:         1,
	}

	actualReverse := tuple.Reverse()
	expectedReverse := icmpTuple{
		icmpVersion: 4,
		srcIP:       net.IPv4(192, 168, 0, 2),
		dstIP:       net.IPv4(192, 168, 0, 1),
		id:          256,
		seq:         1,
	}

	assert.Equal(t, expectedReverse, actualReverse)
}

func BenchmarkIcmpTupleReverse(b *testing.B) {
	tuple := icmpTuple{
		icmpVersion: 4,
		srcIP:       net.IPv4(192, 168, 0, 1),
		dstIP:       net.IPv4(192, 168, 0, 2),
		id:          256,
		seq:         1,
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		tuple.Reverse()
	}
}

func TestIcmpTupleHashable(t *testing.T) {
	tuple := icmpTuple{
		icmpVersion: 4,
		srcIP:       net.IPv4(192, 168, 0, 1),
		dstIP:       net.IPv4(192, 168, 0, 2),
		id:          256,
		seq:         1,
	}

	actualHashable := tuple.Hashable()
	expectedHashable := hashableIcmpTuple{
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 192, 168, 0, 1,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 192, 168, 0, 2,
		1, 0,
		0, 1,
		4,
	}

	assert.Equal(t, expectedHashable, actualHashable)
}

func BenchmarkIcmpTupleHashable(b *testing.B) {
	tuple := icmpTuple{
		icmpVersion: 4,
		srcIP:       net.IPv4(192, 168, 0, 1),
		dstIP:       net.IPv4(192, 168, 0, 2),
		id:          256,
		seq:         1,
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		tuple.Hashable()
	}
}

func TestIcmpTupleToString(t *testing.T) {
	tuple := icmpTuple{
		icmpVersion: 4,
		srcIP:       net.IPv4(192, 168, 0, 1),
		dstIP:       net.IPv4(192, 168, 0, 2),
		id:          256,
		seq:         1,
	}

	actualString := tuple.String()
	expectedString := "icmpTuple version[4] src[192.168.0.1] dst[192.168.0.2] id[256] seq[1]"

	assert.Equal(t, expectedString, actualString)
}
