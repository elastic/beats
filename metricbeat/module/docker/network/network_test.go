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

package network

import (
	"testing"
	"time"
)

var oldNetRaw = make([]NetRaw, 3)
var newNetRaw = make([]NetRaw, 3)
var netService = &NetService{
	NetworkStatPerContainer: make(map[string]map[string]NetRaw),
}

func TestGetRxBytesPerSecond(t *testing.T) {
	oldRxBytes := []uint64{20, 0, 210}
	newRxBytes := []uint64{120, 0, 103}
	for index := range oldNetRaw {
		setTime(index)
		oldNetRaw[index].RxBytes = oldRxBytes[index]
		newNetRaw[index].RxBytes = newRxBytes[index]
	}
	rxBytesTest := []struct {
		givenOld NetRaw
		givenNew NetRaw
		expected float64
	}{
		{oldNetRaw[0], newNetRaw[0], 50},
		{oldNetRaw[1], newNetRaw[1], 0},
		{oldNetRaw[2], newNetRaw[2], 0},
	}
	for _, tt := range rxBytesTest {
		out := netService.getRxBytesPerSecond(&tt.givenNew, &tt.givenOld)
		if out != tt.expected {
			t.Errorf("getRxBytesPerSecond(%v,%v) => %v, want %v", tt.givenNew, tt.givenOld, out, tt.expected)
		}
	}
}

func TestGetRxDroppedPerSeconde(t *testing.T) {
	oldRxDroppedBytes := []uint64{40, 645789, 0}
	newRxDroppedBytes := []uint64{240, 12345, 0}
	for index := range oldNetRaw {
		setTime(index)
		oldNetRaw[index].RxDropped = oldRxDroppedBytes[index]
		newNetRaw[index].RxDropped = newRxDroppedBytes[index]
	}
	rxDroppedTest := []struct {
		givenOld NetRaw
		givenNew NetRaw
		expected float64
	}{
		{oldNetRaw[0], newNetRaw[0], 100},
		{oldNetRaw[1], newNetRaw[1], 0},
		{oldNetRaw[2], newNetRaw[2], 0},
	}
	for _, tt := range rxDroppedTest {
		out := netService.getRxDroppedPerSecond(&tt.givenNew, &tt.givenOld)
		if out != tt.expected {
			t.Errorf("getRxDroppedPerSecond(%v,%v) => %v, want %v", tt.givenNew, tt.givenOld, out, tt.expected)
		}
	}
}

func TestGetRxPacketsPerSeconde(t *testing.T) {
	oldRxPacketsBytes := []uint64{40, 265, 0}
	newRxPacketsBytes := []uint64{140, 26, 0}
	for index := range oldNetRaw {
		setTime(index)
		oldNetRaw[index].RxPackets = oldRxPacketsBytes[index]
		newNetRaw[index].RxPackets = newRxPacketsBytes[index]
	}
	rxPacketTest := []struct {
		givenOld NetRaw
		givenNew NetRaw
		expected float64
	}{
		{oldNetRaw[0], newNetRaw[0], 50},
		{oldNetRaw[1], newNetRaw[1], 0},
		{oldNetRaw[2], newNetRaw[2], 0},
	}
	for _, tt := range rxPacketTest {
		out := netService.getRxPacketsPerSecond(&tt.givenNew, &tt.givenOld)
		if out != tt.expected {
			t.Errorf("getRxPacketsPerSecond(%v,%v) => %v, want %v", tt.givenNew, tt.givenOld, out, tt.expected)
		}
	}
}

func TestGetRxErrorsPerSeconde(t *testing.T) {
	oldRxErrorsBytes := []uint64{0, 150, 986}
	newRxErrorsBytes := []uint64{0, 1150, 653}
	for index := range oldNetRaw {
		setTime(index)
		oldNetRaw[index].RxErrors = oldRxErrorsBytes[index]
		newNetRaw[index].RxErrors = newRxErrorsBytes[index]
	}
	rxPacketTest := []struct {
		givenOld NetRaw
		givenNew NetRaw
		expected float64
	}{
		{oldNetRaw[0], newNetRaw[0], 0},
		{oldNetRaw[1], newNetRaw[1], 500},
		{oldNetRaw[2], newNetRaw[2], 0},
	}
	for _, tt := range rxPacketTest {
		out := netService.getRxErrorsPerSecond(&tt.givenNew, &tt.givenOld)
		if out != tt.expected {
			t.Errorf("getRxErrorsPerSecond(%v,%v) => %v, want %v", tt.givenNew, tt.givenOld, out, tt.expected)
		}
	}
}

func TestGetTxBytesPerSecond(t *testing.T) {
	oldTxBytes := []uint64{0, 995, 986}
	newTxBytes := []uint64{0, 2995, 653}
	for index := range oldNetRaw {
		setTime(index)
		oldNetRaw[index].TxBytes = oldTxBytes[index]
		newNetRaw[index].TxBytes = newTxBytes[index]
	}
	txBytesTest := []struct {
		givenOld NetRaw
		givenNew NetRaw
		expected float64
	}{
		{oldNetRaw[0], newNetRaw[0], 0},
		{oldNetRaw[1], newNetRaw[1], 1000},
		{oldNetRaw[2], newNetRaw[2], 0},
	}
	for _, tt := range txBytesTest {
		out := netService.getTxBytesPerSecond(&tt.givenNew, &tt.givenOld)
		if out != tt.expected {
			t.Errorf("getTxBytesPerSecond(%v,%v) => %v, want %v", tt.givenNew, tt.givenOld, out, tt.expected)
		}
	}
}

func TestGetTxDroppedPerSeconde(t *testing.T) {
	oldTxDropped := []uint64{0, 5, 1236}
	newTxDropped := []uint64{0, 15, 569}
	for index := range oldNetRaw {
		setTime(index)
		oldNetRaw[index].TxDropped = oldTxDropped[index]
		newNetRaw[index].TxDropped = newTxDropped[index]
	}
	txDroppedTest := []struct {
		givenOld NetRaw
		givenNew NetRaw
		expected float64
	}{
		{oldNetRaw[0], newNetRaw[0], 0},
		{oldNetRaw[1], newNetRaw[1], 5},
		{oldNetRaw[2], newNetRaw[2], 0},
	}
	for _, tt := range txDroppedTest {
		out := netService.getTxDroppedPerSecond(&tt.givenNew, &tt.givenOld)
		if out != tt.expected {
			t.Errorf("getTxDroppedPerSecond(%v,%v) => %v, want %v", tt.givenNew, tt.givenOld, out, tt.expected)
		}
	}
}

func TestGetTxPacketsPerSeconde(t *testing.T) {
	oldTxPacket := []uint64{102, 52, 0}
	newTxPacket := []uint64{2102, 15, 0}
	for index := range oldNetRaw {
		setTime(index)
		oldNetRaw[index].TxPackets = oldTxPacket[index]
		newNetRaw[index].TxPackets = newTxPacket[index]
	}
	txPacketTest := []struct {
		givenOld NetRaw
		givenNew NetRaw
		expected float64
	}{
		{oldNetRaw[0], newNetRaw[0], 1000},
		{oldNetRaw[1], newNetRaw[1], 0},
		{oldNetRaw[2], newNetRaw[2], 0},
	}
	for _, tt := range txPacketTest {
		out := netService.getTxPacketsPerSecond(&tt.givenNew, &tt.givenOld)
		if out != tt.expected {
			t.Errorf("getTxPacketsPerSecond(%v,%v) => %v, want %v", tt.givenNew, tt.givenOld, out, tt.expected)
		}
	}
}

func TestGetTxErrorsPerSecond(t *testing.T) {
	oldTxErrors := []uint64{995, 0, 30}
	newTxErrors := []uint64{1995, 0, 10}
	for index := range oldNetRaw {
		setTime(index)
		oldNetRaw[index].TxErrors = oldTxErrors[index]
		newNetRaw[index].TxErrors = newTxErrors[index]
	}
	txErrorsTest := []struct {
		givenOld NetRaw
		givenNew NetRaw
		expected float64
	}{
		{oldNetRaw[0], newNetRaw[0], 500},
		{oldNetRaw[1], newNetRaw[1], 0},
		{oldNetRaw[2], newNetRaw[2], 0},
	}
	for _, tt := range txErrorsTest {
		out := netService.getTxErrorsPerSecond(&tt.givenNew, &tt.givenOld)
		if out != tt.expected {
			t.Errorf("getTxErrorsPerSecond(%v,%v) => %v, want %v", tt.givenNew, tt.givenOld, out, tt.expected)
		}
	}
}

func setTime(index int) {
	oldNetRaw[index].Time = time.Now()
	newNetRaw[index].Time = oldNetRaw[index].Time.Add(time.Duration(2000000000))
}
