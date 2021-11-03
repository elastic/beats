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

package file_integrity

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/elastic/beats/v7/libbeat/common"
)

func TestExeObjParser(t *testing.T) {
	const (
		pkg    = "./testdata"
		target = "./testdata/executable"
	)

	flagSets := map[string][]string{
		"go":     nil,
		"garble": {"-literals", "-tiny"},
	}

	for _, platform := range []struct {
		goos, format string
	}{
		{"linux", "elf"},
		{"darwin", "macho"},
		{"plan9", "plan9"},
		{"windows", "pe"},
	} {
		for _, builder := range []string{
			"go",
			"garble",
		} {
			flags := flagSets[builder]
			cmd, err := build(platform.goos, builder, pkg, target, flags)
			if err != nil {
				t.Errorf("failed to build test for GOOS=%s %s: %v",
					platform.goos, cmd, err)
				continue
			}

			key := fmt.Sprintf("GOOS=%s %s build", platform.goos, builder)
			if flags != nil {
				key += " " + strings.Join(flags, " ")
			}
			t.Run(fmt.Sprintf("executableObject_%s_%s_%v", platform.goos, builder, strings.Join(flags, "_")), func(t *testing.T) {
				got := make(common.MapStr)
				err := exeObjParser(nil).Parse(got, target)
				if err != nil {
					t.Errorf("unexpected error calling exeObjParser.Parse: %v", err)
				}

				fields := []struct {
					path string
					cmp  func(a, b interface{}) bool
				}{
					{path: "import_hash", cmp: func(a, b interface{}) bool { return fmt.Sprint(a) == fmt.Sprint(b) }},
					{path: "imphash", cmp: func(a, b interface{}) bool { return fmt.Sprint(a) == fmt.Sprint(b) }},
					{path: "symhash", cmp: func(a, b interface{}) bool { return fmt.Sprint(a) == fmt.Sprint(b) }},
					{path: "imports", cmp: approxImports(platform.goos, builder)},
					{path: "imports_names_entropy", cmp: approxFloat64(1)},
					{path: "go_import_hash", cmp: approxHash(platform.goos, builder)},
					{path: "go_imports", cmp: approxImports(platform.goos, builder)},
					{path: "go_imports_names_entropy", cmp: approxFloat64(1)},
					{path: "go_stripped", cmp: func(a, b interface{}) bool { return a == b }},
					{path: "sections", cmp: approxSections(0.5)},
				}

				for _, f := range fields {
					path := platform.format + "." + f.path
					wantV, wantErr := want[key].GetValue(path)
					gotV, gotErr := got.GetValue(path)
					if gotErr != wantErr {
						t.Errorf("unexpected error for %s %s: got:%v want:%v", key, path, gotErr, wantErr)
					}
					if gotV == nil {
						continue
					}
					if !f.cmp(gotV, wantV) {
						t.Errorf("unexpected result for %s %s:\ngot: %v\nwant:%v", key, path, gotV, wantV)
					}
				}
			})
		}
	}
	os.Remove(target)
}

func build(goos, builder, path, target string, flags []string) (*exec.Cmd, error) {
	cmd := exec.Command(builder, append(flags[:len(flags):len(flags)], "build", "-o", target, path)...)
	cmd.Env = append([]string{"GOOS=" + goos}, os.Environ()...)
	cmd.Stderr = os.Stderr
	return cmd, cmd.Run()
}

func approxHash(goos, builder string) func(a, b interface{}) bool {
	return func(a, b interface{}) bool {
		as := fmt.Sprint(a)
		bs := fmt.Sprint(b)
		if len(as) != len(bs) {
			return false
		}
		if goos == "darwin" && builder == "garble" {
			// We can't know more since the hash depends on Go version.
			return true
		}
		return as == bs
	}
}

func approxImports(goos, builder string) func(a, b interface{}) bool {
	return func(a, b interface{}) bool {
		as, ok := a.([]string)
		if !ok {
			return false
		}
		bs, ok := b.([]string)
		if !ok {
			return false
		}
		if goos == "darwin" && builder == "garble" {
			// We can't know more since the symbols depend on Go version.
			return true
		}
		if len(as) != len(bs) {
			return false
		}
		return reflect.DeepEqual(as, bs)
	}
}

func approxFloat64(tol float64) func(a, b interface{}) bool {
	return func(a, b interface{}) bool {
		af, ok := a.(float64)
		if !ok {
			return false
		}
		bf, ok := b.(float64)
		if !ok {
			return false
		}
		return math.Abs(af-bf) <= tol
	}
}

func approxSections(tol float64) func(a, b interface{}) bool {
	return func(a, b interface{}) bool {
		aObj, ok := a.([]objSection)
		if !ok {
			return false
		}
		bObj, ok := b.([]objSection)
		if !ok {
			return false
		}
		if len(aObj) != len(bObj) {
			return false
		}
		for i := range aObj {
			if (aObj[i].Name == nil) != (bObj[i].Name == nil) || (aObj[i].Name != nil && *aObj[i].Name != *bObj[i].Name) {
				return false
			}
			if (aObj[i].Size == nil && *aObj[i].Size == 0) != (bObj[i].Size == nil && *bObj[i].Size == 0) {
				return false
			}
			if ((aObj[i].Entropy == nil) != (bObj[i].Entropy == nil)) || (aObj[i].Entropy != nil && math.Abs(*aObj[i].Entropy-*bObj[i].Entropy) > tol) {
				return false
			}
		}
		return true
	}
}

func strPtr(s string) *string       { return &s }
func float64Ptr(f float64) *float64 { return &f }
func uint64Ptr(u uint64) *uint64    { return &u }

func (o objSection) String() string {
	name := "<nil>"
	if o.Name != nil {
		name = *o.Name
	}
	size := "<nil>"
	if o.Size != nil {
		size = strconv.FormatUint(*o.Size, 16)
	}
	entropy := "<nil>"
	if o.Entropy != nil {
		entropy = strconv.FormatFloat(*o.Entropy, 'f', -1, 64)
	}
	return fmt.Sprintf("{Name: %q, Size: %s, Entropy: %s}", name, size, entropy)
}

var want = map[string]common.MapStr{
	"GOOS=plan9 garble build -literals -tiny": {
		"plan9": common.MapStr{
			"go_import_hash": "d41d8cd98f00b204e9800998ecf8427e",
			"go_stripped":    true,
			"sections": []objSection{
				{Name: strPtr("text"), Size: uint64Ptr(0xfcd30), Entropy: float64Ptr(5.864881580649509)},
				{Name: strPtr("data"), Size: uint64Ptr(0x16d80), Entropy: float64Ptr(4.651998848388147)},
				{Name: strPtr("syms"), Size: uint64Ptr(0x0), Entropy: float64Ptr(0.0)},
				{Name: strPtr("spsz"), Size: uint64Ptr(0x0), Entropy: float64Ptr(0.0)},
				{Name: strPtr("pcsz"), Size: uint64Ptr(0x0), Entropy: float64Ptr(0.0)},
			},
		},
	},
	"GOOS=windows go build": {
		"pe": common.MapStr{
			"imphash":                  "c7269d59926fa4252270f407e4dab043",
			"go_import_hash":           "10bddcb4cee42080f76c88d9ff964491",
			"go_imports_names_entropy": 4.156563879566413,
			"go_stripped":              false,
			"sections": []objSection{
				{Name: strPtr(".text"), Size: uint64Ptr(0x8e400), Entropy: float64Ptr(6.1762021221116195)},
				{Name: strPtr(".rdata"), Size: uint64Ptr(0x9ea00), Entropy: float64Ptr(5.139459498570865)},
				{Name: strPtr(".data"), Size: uint64Ptr(0x17a00), Entropy: float64Ptr(4.60076261878884)},
				{Name: strPtr(".zdebug_abbrev"), Size: uint64Ptr(0x200), Entropy: float64Ptr(4.8292159200679565)},
				{Name: strPtr(".zdebug_line"), Size: uint64Ptr(0x1cc00), Entropy: float64Ptr(7.992389830575166)},
				{Name: strPtr(".zdebug_frame"), Size: uint64Ptr(0x5800), Entropy: float64Ptr(7.926764090429505)},
				{Name: strPtr(".debug_gdb_scripts"), Size: uint64Ptr(0x200), Entropy: float64Ptr(0.8418026665453624)},
				{Name: strPtr(".zdebug_info"), Size: uint64Ptr(0x32a00), Entropy: float64Ptr(7.996530718084179)},
				{Name: strPtr(".zdebug_loc"), Size: uint64Ptr(0x1ba00), Entropy: float64Ptr(7.989774523689841)},
				{Name: strPtr(".zdebug_ranges"), Size: uint64Ptr(0x9600), Entropy: float64Ptr(7.783964713570338)},
				{Name: strPtr(".idata"), Size: uint64Ptr(0x600), Entropy: float64Ptr(3.61484457240618)},
				{Name: strPtr(".reloc"), Size: uint64Ptr(0x6a00), Entropy: float64Ptr(5.441393317161353)},
				{Name: strPtr(".symtab"), Size: uint64Ptr(0x17a00), Entropy: float64Ptr(5.120436252433234)},
			},
			"import_hash": "c7269d59926fa4252270f407e4dab043",
			"imports": []string{
				"kernel32.writefile",
				"kernel32.writeconsolew",
				"kernel32.waitformultipleobjects",
				"kernel32.waitforsingleobject",
				"kernel32.virtualquery",
				"kernel32.virtualfree",
				"kernel32.virtualalloc",
				"kernel32.switchtothread",
				"kernel32.suspendthread",
				"kernel32.sleep",
				"kernel32.setwaitabletimer",
				"kernel32.setunhandledexceptionfilter",
				"kernel32.setprocesspriorityboost",
				"kernel32.setevent",
				"kernel32.seterrormode",
				"kernel32.setconsolectrlhandler",
				"kernel32.resumethread",
				"kernel32.postqueuedcompletionstatus",
				"kernel32.loadlibrarya",
				"kernel32.loadlibraryw",
				"kernel32.setthreadcontext",
				"kernel32.getthreadcontext",
				"kernel32.getsysteminfo",
				"kernel32.getsystemdirectorya",
				"kernel32.getstdhandle",
				"kernel32.getqueuedcompletionstatusex",
				"kernel32.getprocessaffinitymask",
				"kernel32.getprocaddress",
				"kernel32.getenvironmentstringsw",
				"kernel32.getconsolemode",
				"kernel32.freeenvironmentstringsw",
				"kernel32.exitprocess",
				"kernel32.duplicatehandle",
				"kernel32.createwaitabletimerexw",
				"kernel32.createthread",
				"kernel32.createiocompletionport",
				"kernel32.createfilea",
				"kernel32.createeventa",
				"kernel32.closehandle",
				"kernel32.addvectoredexceptionhandler",
			},
			"imports_names_entropy": 4.2079021689106195,
			"go_imports": []string{
				"github.com/elastic/beats/v7/auditbeat/module/file_integrity/testdata/b.Used",
				"github.com/elastic/beats/v7/auditbeat/module/file_integrity/testdata/b.hash",
			},
		},
	},
	"GOOS=windows garble build -literals -tiny": {
		"pe": common.MapStr{
			"import_hash": "c7269d59926fa4252270f407e4dab043",
			"imphash":     "c7269d59926fa4252270f407e4dab043",
			"imports": []string{
				"kernel32.writefile",
				"kernel32.writeconsolew",
				"kernel32.waitformultipleobjects",
				"kernel32.waitforsingleobject",
				"kernel32.virtualquery",
				"kernel32.virtualfree",
				"kernel32.virtualalloc",
				"kernel32.switchtothread",
				"kernel32.suspendthread",
				"kernel32.sleep",
				"kernel32.setwaitabletimer",
				"kernel32.setunhandledexceptionfilter",
				"kernel32.setprocesspriorityboost",
				"kernel32.setevent",
				"kernel32.seterrormode",
				"kernel32.setconsolectrlhandler",
				"kernel32.resumethread",
				"kernel32.postqueuedcompletionstatus",
				"kernel32.loadlibrarya",
				"kernel32.loadlibraryw",
				"kernel32.setthreadcontext",
				"kernel32.getthreadcontext",
				"kernel32.getsysteminfo",
				"kernel32.getsystemdirectorya",
				"kernel32.getstdhandle",
				"kernel32.getqueuedcompletionstatusex",
				"kernel32.getprocessaffinitymask",
				"kernel32.getprocaddress",
				"kernel32.getenvironmentstringsw",
				"kernel32.getconsolemode",
				"kernel32.freeenvironmentstringsw",
				"kernel32.exitprocess",
				"kernel32.duplicatehandle",
				"kernel32.createwaitabletimerexw",
				"kernel32.createthread",
				"kernel32.createiocompletionport",
				"kernel32.createfilea",
				"kernel32.createeventa",
				"kernel32.closehandle",
				"kernel32.addvectoredexceptionhandler",
			},
			"imports_names_entropy": 4.2079021689106195,
			"go_import_hash":        "d41d8cd98f00b204e9800998ecf8427e",
			"go_stripped":           true,
			"sections": []objSection{
				{Name: strPtr(".text"), Size: uint64Ptr(0x83000), Entropy: float64Ptr(6.1836267499241755)},
				{Name: strPtr(".rdata"), Size: uint64Ptr(0x97a00), Entropy: float64Ptr(5.103956377519968)},
				{Name: strPtr(".data"), Size: uint64Ptr(0x17800), Entropy: float64Ptr(4.609734714924862)},
				{Name: strPtr(".idata"), Size: uint64Ptr(0x600), Entropy: float64Ptr(3.6082349091764896)},
				{Name: strPtr(".reloc"), Size: uint64Ptr(0x6800), Entropy: float64Ptr(5.428635483932532)},
				{Name: strPtr(".symtab"), Size: uint64Ptr(0x200), Entropy: float64Ptr(0.020393135236084953)},
			},
		},
	},
	"GOOS=linux go build": {
		"elf": common.MapStr{
			"go_imports_names_entropy": 4.156563879566413,
			"go_stripped":              false,
			"sections": []objSection{
				{Name: strPtr(""), Size: uint64Ptr(0x0), Entropy: float64Ptr(0.0)},
				{Name: strPtr(".text"), Size: uint64Ptr(0x7ffd6), Entropy: float64Ptr(6.171439471921469)},
				{Name: strPtr(".rodata"), Size: uint64Ptr(0x35940), Entropy: float64Ptr(4.355815364247477)},
				{Name: strPtr(".shstrtab"), Size: uint64Ptr(0x17a), Entropy: float64Ptr(4.332514286812164)},
				{Name: strPtr(".typelink"), Size: uint64Ptr(0x4f0), Entropy: float64Ptr(3.7700952245237285)},
				{Name: strPtr(".itablink"), Size: uint64Ptr(0x60), Entropy: float64Ptr(2.149135857994785)},
				{Name: strPtr(".gosymtab"), Size: uint64Ptr(0x0), Entropy: float64Ptr(0.0)},
				{Name: strPtr(".gopclntab"), Size: uint64Ptr(0x5a5c8), Entropy: float64Ptr(5.486008546433886)},
				{Name: strPtr(".go.buildinfo"), Size: uint64Ptr(0x20), Entropy: float64Ptr(3.560820381093429)},
				{Name: strPtr(".noptrdata"), Size: uint64Ptr(0x10720), Entropy: float64Ptr(5.6078976935228955)},
				{Name: strPtr(".data"), Size: uint64Ptr(0x7810), Entropy: float64Ptr(1.6046396762408546)},
				{Name: strPtr(".bss"), Size: uint64Ptr(0x2ef48), Entropy: float64Ptr(7.994394841911349)},
				{Name: strPtr(".noptrbss"), Size: uint64Ptr(0x5360), Entropy: float64Ptr(7.975914457802507)},
				{Name: strPtr(".zdebug_abbrev"), Size: uint64Ptr(0x119), Entropy: float64Ptr(7.186678878967747)},
				{Name: strPtr(".zdebug_line"), Size: uint64Ptr(0x1b90f), Entropy: float64Ptr(7.991063018317068)},
				{Name: strPtr(".zdebug_frame"), Size: uint64Ptr(0x551b), Entropy: float64Ptr(7.925008509003898)},
				{Name: strPtr(".debug_gdb_scripts"), Size: uint64Ptr(0x31), Entropy: float64Ptr(4.249529170858451)},
				{Name: strPtr(".zdebug_info"), Size: uint64Ptr(0x31a2a), Entropy: float64Ptr(7.995374455849462)},
				{Name: strPtr(".zdebug_loc"), Size: uint64Ptr(0x198d9), Entropy: float64Ptr(7.988800696836627)},
				{Name: strPtr(".zdebug_ranges"), Size: uint64Ptr(0x8fbc), Entropy: float64Ptr(7.7864300204494885)},
				{Name: strPtr(".note.go.buildid"), Size: uint64Ptr(0x64), Entropy: float64Ptr(5.3883674395583805)},
				{Name: strPtr(".symtab"), Size: uint64Ptr(0xc5e8), Entropy: float64Ptr(3.2101068454851)},
				{Name: strPtr(".strtab"), Size: uint64Ptr(0xb2d6), Entropy: float64Ptr(4.811971045761911)},
			},
			"import_hash":    "d41d8cd98f00b204e9800998ecf8427e",
			"go_import_hash": "10bddcb4cee42080f76c88d9ff964491",
			"go_imports": []string{
				"github.com/elastic/beats/v7/auditbeat/module/file_integrity/testdata/b.Used",
				"github.com/elastic/beats/v7/auditbeat/module/file_integrity/testdata/b.hash",
			},
		},
	},
	"GOOS=linux garble build -literals -tiny": {
		"elf": common.MapStr{
			"import_hash":    "d41d8cd98f00b204e9800998ecf8427e",
			"go_import_hash": "d41d8cd98f00b204e9800998ecf8427e",
			"go_stripped":    true,
			"sections": []objSection{
				{Name: strPtr(""), Size: uint64Ptr(0x0), Entropy: float64Ptr(0.0)},
				{Name: strPtr(".text"), Size: uint64Ptr(0x74f85), Entropy: float64Ptr(6.180689292506103)},
				{Name: strPtr(".rodata"), Size: uint64Ptr(0x331e4), Entropy: float64Ptr(4.256094583892582)},
				{Name: strPtr(".shstrtab"), Size: uint64Ptr(0x94), Entropy: float64Ptr(4.278922006970282)},
				{Name: strPtr(".typelink"), Size: uint64Ptr(0x4ec), Entropy: float64Ptr(3.6948502589031187)},
				{Name: strPtr(".itablink"), Size: uint64Ptr(0x60), Entropy: float64Ptr(2.142816541821228)},
				{Name: strPtr(".gosymtab"), Size: uint64Ptr(0x0), Entropy: float64Ptr(0.0)},
				{Name: strPtr(".gopclntab"), Size: uint64Ptr(0x56370), Entropy: float64Ptr(5.428192633737662)},
				{Name: strPtr(".go.buildinfo"), Size: uint64Ptr(0x20), Entropy: float64Ptr(3.560820381093429)},
				{Name: strPtr(".noptrdata"), Size: uint64Ptr(0x10720), Entropy: float64Ptr(5.606503969275871)},
				{Name: strPtr(".data"), Size: uint64Ptr(0x7570), Entropy: float64Ptr(1.5492706480530087)},
				{Name: strPtr(".bss"), Size: uint64Ptr(0x2ef48), Entropy: float64Ptr(0.0)},
				{Name: strPtr(".noptrbss"), Size: uint64Ptr(0x5340), Entropy: float64Ptr(0.0)},
			},
		},
	},
	"GOOS=darwin go build": {
		"macho": common.MapStr{
			"symhash": "d3ccf195b62a9279c3c19af1080497ec",
			"imports": []string{
				"___error",
				"__exit",
				"_clock_gettime",
				"_close",
				"_closedir",
				"_execve",
				"_fcntl",
				"_fstat64",
				"_getcwd",
				"_getpid",
				"_kevent",
				"_kill",
				"_kqueue",
				"_lseek",
				"_mach_absolute_time",
				"_mach_timebase_info",
				"_madvise",
				"_mmap",
				"_munmap",
				"_open",
				"_pipe",
				"_pthread_attr_getstacksize",
				"_pthread_attr_init",
				"_pthread_attr_setdetachstate",
				"_pthread_cond_init",
				"_pthread_cond_signal",
				"_pthread_cond_timedwait_relative_np",
				"_pthread_cond_wait",
				"_pthread_create",
				"_pthread_kill",
				"_pthread_mutex_init",
				"_pthread_mutex_lock",
				"_pthread_mutex_unlock",
				"_pthread_self",
				"_pthread_sigmask",
				"_raise",
				"_read",
				"_sigaction",
				"_sigaltstack",
				"_stat64",
				"_sysctl",
				"_usleep",
				"_write",
			},
			"go_imports": []string{
				"github.com/elastic/beats/v7/auditbeat/module/file_integrity/testdata/b.Used",
				"github.com/elastic/beats/v7/auditbeat/module/file_integrity/testdata/b.hash",
			},
			"sections": []objSection{
				{Name: strPtr("__text"), Size: uint64Ptr(0x8be36), Entropy: float64Ptr(6.165891284878783)},
				{Name: strPtr("__symbol_stub1"), Size: uint64Ptr(0x102), Entropy: float64Ptr(3.6276890098831442)},
				{Name: strPtr("__rodata"), Size: uint64Ptr(0x38b4f), Entropy: float64Ptr(4.379438168970728)},
				{Name: strPtr("__typelink"), Size: uint64Ptr(0x550), Entropy: float64Ptr(3.6495169670279197)},
				{Name: strPtr("__itablink"), Size: uint64Ptr(0x78), Entropy: float64Ptr(2.6320431334452543)},
				{Name: strPtr("__gosymtab"), Size: uint64Ptr(0x0), Entropy: float64Ptr(0.0)},
				{Name: strPtr("__gopclntab"), Size: uint64Ptr(0x614a0), Entropy: float64Ptr(5.469825725243185)},
				{Name: strPtr("__go_buildinfo"), Size: uint64Ptr(0x20), Entropy: float64Ptr(3.7959585933443494)},
				{Name: strPtr("__nl_symbol_ptr"), Size: uint64Ptr(0x158), Entropy: float64Ptr(0.0)},
				{Name: strPtr("__noptrdata"), Size: uint64Ptr(0x10780), Entropy: float64Ptr(5.599986219664661)},
				{Name: strPtr("__data"), Size: uint64Ptr(0x7470), Entropy: float64Ptr(1.743720912770626)},
				{Name: strPtr("__bss"), Size: uint64Ptr(0x2f068), Entropy: float64Ptr(6.139244424626403)},
				{Name: strPtr("__noptrbss"), Size: uint64Ptr(0x51c0), Entropy: float64Ptr(5.658961220743402)},
				{Name: strPtr("__zdebug_abbrev"), Size: uint64Ptr(0x117), Entropy: float64Ptr(7.166065824433164)},
				{Name: strPtr("__zdebug_line"), Size: uint64Ptr(0x1d615), Entropy: float64Ptr(7.991223121221244)},
				{Name: strPtr("__zdebug_frame"), Size: uint64Ptr(0x5b82), Entropy: float64Ptr(7.928371655053322)},
				{Name: strPtr("__debug_gdb_scri"), Size: uint64Ptr(0x31), Entropy: float64Ptr(4.249529170858451)},
				{Name: strPtr("__zdebug_info"), Size: uint64Ptr(0x33a7b), Entropy: float64Ptr(7.996688904672549)},
				{Name: strPtr("__zdebug_loc"), Size: uint64Ptr(0x1a57f), Entropy: float64Ptr(7.983862888050245)},
				{Name: strPtr("__zdebug_ranges"), Size: uint64Ptr(0x8371), Entropy: float64Ptr(7.891741786242217)},
			},
			"import_hash":              "d3ccf195b62a9279c3c19af1080497ec",
			"imports_names_entropy":    4.132925542571368,
			"go_import_hash":           "10bddcb4cee42080f76c88d9ff964491",
			"go_imports_names_entropy": 4.156563879566413,
			"go_stripped":              false,
		},
	},
	"GOOS=darwin garble build -literals -tiny": {
		"macho": common.MapStr{
			"sections": []objSection{
				{Name: strPtr("__text"), Size: uint64Ptr(0x80e52), Entropy: float64Ptr(6.170434297924308)},
				{Name: strPtr("__symbol_stub1"), Size: uint64Ptr(0x102), Entropy: float64Ptr(3.5781727974012107)},
				{Name: strPtr("__rodata"), Size: uint64Ptr(0x367b3), Entropy: float64Ptr(4.266589957262039)},
				{Name: strPtr("__typelink"), Size: uint64Ptr(0x554), Entropy: float64Ptr(3.72866886425051)},
				{Name: strPtr("__itablink"), Size: uint64Ptr(0x78), Entropy: float64Ptr(2.6182943321190826)},
				{Name: strPtr("__gosymtab"), Size: uint64Ptr(0x0), Entropy: float64Ptr(0.0)},
				{Name: strPtr("__gopclntab"), Size: uint64Ptr(0x5cf68), Entropy: float64Ptr(5.416616743010021)},
				{Name: strPtr("__go_buildinfo"), Size: uint64Ptr(0x20), Entropy: float64Ptr(3.8584585933443494)},
				{Name: strPtr("__nl_symbol_ptr"), Size: uint64Ptr(0x158), Entropy: float64Ptr(0.0)},
				{Name: strPtr("__noptrdata"), Size: uint64Ptr(0x10780), Entropy: float64Ptr(5.599921142740692)},
				{Name: strPtr("__data"), Size: uint64Ptr(0x71f0), Entropy: float64Ptr(1.7218753759505945)},
				{Name: strPtr("__bss"), Size: uint64Ptr(0x2f088), Entropy: float64Ptr(6.135077333312854)},
				{Name: strPtr("__noptrbss"), Size: uint64Ptr(0x51a0), Entropy: float64Ptr(5.558987105806822)},
			},
			"import_hash": "d3ccf195b62a9279c3c19af1080497ec",
			"imports": []string{
				"___error",
				"__exit",
				"_clock_gettime",
				"_close",
				"_closedir",
				"_execve",
				"_fcntl",
				"_fstat64",
				"_getcwd",
				"_getpid",
				"_kevent",
				"_kill",
				"_kqueue",
				"_lseek",
				"_mach_absolute_time",
				"_mach_timebase_info",
				"_madvise",
				"_mmap",
				"_munmap",
				"_open",
				"_pipe",
				"_pthread_attr_getstacksize",
				"_pthread_attr_init",
				"_pthread_attr_setdetachstate",
				"_pthread_cond_init",
				"_pthread_cond_signal",
				"_pthread_cond_timedwait_relative_np",
				"_pthread_cond_wait",
				"_pthread_create",
				"_pthread_kill",
				"_pthread_mutex_init",
				"_pthread_mutex_lock",
				"_pthread_mutex_unlock",
				"_pthread_self",
				"_pthread_sigmask",
				"_raise",
				"_read",
				"_sigaction",
				"_sigaltstack",
				"_stat64",
				"_sysctl",
				"_usleep",
				"_write",
			},
			"imports_names_entropy": 4.132925542571368,
			"go_imports": []string{
				"evnQ6ZcH.NEfVFrsU",
				"evnQ6ZcH.NEfVFrsU.func1",
				"evnQ6ZcH.obErrEr2",
				"evnQ6ZcH.obErrEr2.func1",
				"evnQ6ZcH.obErrEr2.func1.1",
				"main.main",
				"main.main.func1",
			},
			"symhash":                  "d3ccf195b62a9279c3c19af1080497ec",
			"go_import_hash":           "ea0346ba1d3c7c7e762864b7abd53399",
			"go_imports_names_entropy": 4.416263999653068,
			"go_stripped":              true,
		},
	},
	"GOOS=plan9 go build": {
		"plan9": common.MapStr{
			"go_import_hash": "10bddcb4cee42080f76c88d9ff964491",
			"go_imports": []string{
				"github.com/elastic/beats/v7/auditbeat/module/file_integrity/testdata/b.Used",
				"github.com/elastic/beats/v7/auditbeat/module/file_integrity/testdata/b.hash",
			},
			"go_imports_names_entropy": 4.156563879566413,
			"go_stripped":              false,
			"sections": []objSection{
				{Name: strPtr("text"), Size: uint64Ptr(0x110c88), Entropy: float64Ptr(5.879890481716839)},
				{Name: strPtr("data"), Size: uint64Ptr(0x17000), Entropy: float64Ptr(4.6411984855995065)},
				{Name: strPtr("syms"), Size: uint64Ptr(0xe9fa), Entropy: float64Ptr(5.0968023490458245)},
				{Name: strPtr("spsz"), Size: uint64Ptr(0x0), Entropy: float64Ptr(0.0)},
				{Name: strPtr("pcsz"), Size: uint64Ptr(0x0), Entropy: float64Ptr(0.0)},
			},
		},
	},
}
