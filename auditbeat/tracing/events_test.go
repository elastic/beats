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

package tracing

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

var rawMsg64 = []byte{
	0x9b, 0x05, 0x00, 0x00, 0xae, 0x0e, 0x00, 0x00, 0xa0, 0x52, 0x23, 0xad,
	0xff, 0xff, 0xff, 0xff, 0x3c, 0x00, 0x04, 0x00, 0x03, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x01, 0x7d, 0x56, 0xe6, 0x62, 0xc8, 0x99, 0xc4, 0x25,
	0x73, 0x73, 0x68, 0x64, 0x00, 0x00, 0x00, 0x00,
}

var rawMsg32 = []byte{
	0x9b, 0x05, 0x00, 0x00, 0xae, 0x0e, 0x00, 0x00, 0xa0, 0x52, 0x23, 0xad,
	0x3c, 0x00, 0x04, 0x00, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
	0x7d, 0x56, 0xe6, 0x62, 0xc8, 0x99, 0xc4, 0x25, 0x73, 0x73, 0x68, 0x64,
	0x00, 0x00, 0x00, 0x00,
}

var rawMsg = initRaw()

func initRaw() []byte {
	switch v := unsafe.Sizeof(uintptr(0)); v {
	case 4:
		return rawMsg32
	case 8:
		return rawMsg64
	default:
		panic(v)
	}
}

func BenchmarkMapDecoder(b *testing.B) {
	evs, err := NewTraceFS()
	if err != nil {
		b.Fatal(err)
	}
	probe := Probe{
		Name:      "test_name",
		Address:   "sys_connect",
		Fetchargs: "exe=+0(%ax):string fd=%di:u64 +0(%si):u8 +8(%si):u64 +16(%si):s16 +24(%si):u32",
	}
	err = evs.AddKProbe(probe)
	if err != nil {
		b.Fatal(err)
	}
	desc, err := evs.LoadProbeFormat(probe)
	if err != nil {
		b.Fatal(err)
	}
	decoder := NewMapDecoder(desc)
	b.ResetTimer()
	var sum int
	var meta Metadata
	for i := 0; i < b.N; i++ {
		iface, err := decoder.Decode(rawMsg, meta)
		if err != nil {
			b.Fatal(err)
		}
		m := iface.(map[string]interface{})

		for _, c := range m["exe"].(string) {
			sum += int(c)
		}
		sum += int(m["fd"].(uint64))
		sum += int(m["arg3"].(uint8))
		sum += int(m["arg4"].(uint64))
		sum += int(m["arg5"].(int16))
		sum += int(m["arg6"].(uint32))
	}
	b.StopTimer()
	b.Log("result sum=", sum)
	b.ReportAllocs()
}

func BenchmarkStructDecoder(b *testing.B) {
	evs, err := NewTraceFS()
	if err != nil {
		b.Fatal(err)
	}
	probe := Probe{
		Group:     "test_group",
		Name:      "test_name",
		Address:   "sys_connect",
		Fetchargs: "exe=+0(%ax):string fd=%di:u64 +0(%si):u8 +8(%si):u64 +16(%si):s16 +24(%si):u32",
	}
	err = evs.AddKProbe(probe)
	if err != nil {
		b.Fatal(err)
	}
	desc, err := evs.LoadProbeFormat(probe)
	if err != nil {
		b.Fatal(err)
	}

	type myStruct struct {
		Meta   Metadata `kprobe:"metadata"`
		Type   uint16   `kprobe:"common_type"`
		Flags  uint8    `kprobe:"common_flags"`
		PCount uint8    `kprobe:"common_preempt_count"`
		PID    uint32   `kprobe:"common_pid"`
		IP     uintptr  `kprobe:"__probe_ip"`
		Exe    string   `kprobe:"exe"`
		Fd     uint64   `kprobe:"fd"`
		Arg3   uint8    `kprobe:"arg3"`
		Arg4   uint64   `kprobe:"arg4"`
		Arg5   uint16   `kprobe:"arg5"`
		Arg6   uint32   `kprobe:"arg6"`
	}
	var myAlloc AllocateFn = func() interface{} {
		return new(myStruct)
	}

	decoder, err := NewStructDecoder(desc, myAlloc)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	sum := 0
	var meta Metadata
	for i := 0; i < b.N; i++ {
		iface, err := decoder.Decode(rawMsg, meta)
		if err != nil {
			b.Fatal(err)
		}
		m := iface.(*myStruct)

		for _, c := range m.Exe {
			sum += int(c)
		}
		sum += int(m.Fd)
		sum += int(m.Arg3)
		sum += int(m.Arg4)
		sum += int(m.Arg5)
		sum += int(m.Arg6)
	}
	b.StopTimer()
	b.Log("result sum=", sum)
	b.ReportAllocs()
}

func TestKProbeReal(t *testing.T) {
	// Skipped ...
	t.SkipNow()

	evs, err := NewTraceFS()
	if err != nil {
		t.Fatal(err)
	}
	listAll := func() []Probe {
		list, err := evs.ListKProbes()
		if err != nil {
			t.Fatal(err)
		}
		t.Log("Read ", len(list), "kprobes")
		for idx, probe := range list {
			t.Log(idx, ": ", probe.String())
		}
		return list
	}
	for _, kprobe := range listAll() {
		if err := evs.RemoveKProbe(kprobe); err != nil {
			t.Fatal(err, kprobe.String())
		}
	}
	err = evs.AddKProbe(Probe{
		Name:      "myprobe",
		Address:   "sys_connect",
		Fetchargs: "fd=%di +0(%si) +8(%si) +16(%si) +24(%si)",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = evs.AddKProbe(Probe{
		Type:      TypeKRetProbe,
		Name:      "myretprobe",
		Address:   "do_sys_open",
		Fetchargs: "retval=%ax",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := evs.RemoveAllKProbes(); err != nil {
		t.Fatal(err)
	}
	probe := Probe{
		Name:    "test_kprobe",
		Address: "sys_connect",
		// Fetchargs: "exe=$comm fd=%di +0(%si) +8(%si) +16(%si) +24(%si) +99999(%ax):string",
		Fetchargs: "ax=%ax bx=%bx:u8 cx=%cx:u32 dx=%dx:s16",
	}
	err = evs.AddKProbe(probe)
	if err != nil {
		t.Fatal(err)
	}
	desc, err := evs.LoadProbeFormat(probe)
	if err != nil {
		t.Fatal(err)
	}
	// fmt.Fprintf(os.Stderr, "desc=%+v\n", desc)
	var decoder Decoder
	const useStructDecoder = false
	if useStructDecoder {
		type myStruct struct {
			// Exe string `kprobe:"exe"`
			PID uint32 `kprobe:"common_pid"`
			AX  int64  `kprobe:"ax"`
			BX  uint8  `kprobe:"bx"`
			CX  int32  `kprobe:"cx"`
			DX  uint16 `kprobe:"dx"`
		}
		allocFn := func() interface{} {
			return new(myStruct)
		}
		if decoder, err = NewStructDecoder(desc, allocFn); err != nil {
			t.Fatal(err)
		}
	} else {
		decoder = NewMapDecoder(desc)
	}

	channel, err := NewPerfChannel(WithTimestamp())
	if err != nil {
		t.Fatal(err)
	}

	if err := channel.MonitorProbe(desc, decoder); err != nil {
		t.Fatal(err)
	}

	if err := channel.Run(); err != nil {
		t.Fatal(err)
	}

	timer := time.NewTimer(time.Second * 10)
	defer timer.Stop()

	for active := true; active; {
		select {
		case <-timer.C:
			active = false
		case iface, ok := <-channel.C():
			if !ok {
				active = false
				break
			}
			if true {
				data := iface.(map[string]interface{})
				_, err = fmt.Fprintf(os.Stderr, "Got event len=%d\n", len(data))
				if err != nil {
					panic(err)
				}

				fmt.Fprintf(os.Stderr, "%s event:\n", time.Now().Format(time.RFC3339Nano))
				for k, v := range data {
					fmt.Fprintf(os.Stderr, "    %s: %v\n", k, v)
				}
			}
		case err := <-channel.ErrC():
			t.Log("Err received from channel:", err)
			active = false

		case lost := <-channel.LostC():
			t.Log("lost events:", lost)
		}
	}

	err = channel.Close()
	if err != nil {
		t.Log("channel.Close returned err=", err)
	}

	t.Logf("Got description: %+v", desc)
	err = evs.RemoveKProbe(probe)
	if err != nil {
		t.Fatal(err)
	}
}

func TestKProbeEventsList(t *testing.T) {
	// Make dir to monitor.
	tmpDir, err := os.MkdirTemp("", "events_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.MkdirAll(tmpDir, 0o700); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(tmpDir, "kprobe_events"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	_, err = file.WriteString(`
p:probe_1 fancy_function+0x0 exe=$comm fd=%di:u64 addr=+12(%si):x32
r:kprobe/My-Ret-Probe 0xfff30234111
p:some-other_group/myprobe sys_crash
something wrong here
w:future feature
`)
	if err != nil {
		t.Fatal(err)
	}

	evs, err := NewTraceFSWithPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	kprobes, err := evs.ListKProbes()
	if err != nil {
		panic(err)
	}
	expected := []Probe{
		{
			Type:      TypeKProbe,
			Name:      "probe_1",
			Address:   "fancy_function+0x0",
			Fetchargs: "exe=$comm fd=%di:u64 addr=+12(%si):x32",
		},
		{
			Type:    TypeKRetProbe,
			Group:   "kprobe",
			Name:    "My-Ret-Probe",
			Address: "0xfff30234111",
		},
		{
			Group:   "some-other_group",
			Name:    "myprobe",
			Address: "sys_crash",
		},
	}
	assert.Equal(t, expected, kprobes)
}

func TestKProbeEventsAddRemoveKProbe(t *testing.T) {
	// Make dir to monitor.
	tmpDir, err := os.MkdirTemp("", "events_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.MkdirAll(tmpDir, 0o700); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(tmpDir, "kprobe_events"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	baseContents := `
p:kprobe/existing fancy_function+0x0 exe=$comm fd=%di:u64 addr=+12(%si):x32
r:kprobe/My-Ret-Probe 0xfff30234111
something wrong here
w:future feature
`
	_, err = file.WriteString(baseContents)
	if err != nil {
		t.Fatal(err)
	}

	evs, err := NewTraceFSWithPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	p1 := Probe{Group: "kprobe", Name: "myprobe", Address: "sys_open", Fetchargs: "path=+0(%di):string mode=%si"}
	p2 := Probe{Type: TypeKRetProbe, Name: "myretprobe", Address: "0xffffff123456", Fetchargs: "+0(%di) +8(%di) +16(%di)"}
	assert.NoError(t, evs.AddKProbe(p1))
	assert.NoError(t, evs.AddKProbe(p2))
	assert.NoError(t, evs.RemoveKProbe(p1))
	assert.NoError(t, evs.RemoveKProbe(p2))

	off, err := file.Seek(int64(0), io.SeekStart)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), off)
	contents, err := io.ReadAll(file)
	assert.NoError(t, err)
	expected := append([]byte(baseContents), []byte(
		`p:kprobe/myprobe sys_open path=+0(%di):string mode=%si
r:myretprobe 0xffffff123456 +0(%di) +8(%di) +16(%di)
-:kprobe/myprobe
-:myretprobe
`)...)
	assert.Equal(t, strings.Split(string(expected), "\n"), strings.Split(string(contents), "\n"))
}
