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

//nolint:errorlint,dupl // Bad linters!
package file_integrity

import (
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestExeObjParser(t *testing.T) {
	for _, format := range []string{
		"elf", "macho", "pe",
	} {
		for _, builder := range []string{
			"go",
			"garble",
		} {
			target := fmt.Sprintf("./testdata/%s_%s_executable", builder, format)

			key := fmt.Sprintf("%s_%s", builder, format)
			t.Run(fmt.Sprintf("executableObject_%s_%s", format, builder), func(t *testing.T) {
				if builder == "garble" && format == "pe" {
					t.Skip("skipping test on garbled PE file: see https://github.com/elastic/beats/issues/35705")
				}

				if _, ci := os.LookupEnv("CI"); ci {
					if _, err := os.Stat(target); err != nil && errors.Is(err, fs.ErrNotExist) {
						t.Skip("skipping test because target binary was not found: see https://github.com/elastic/beats/issues/38211")
					}
				}

				got := make(mapstr.M)
				err := exeObjParser(nil).Parse(got, target)
				if err != nil {
					t.Fatalf("unexpected error calling exeObjParser.Parse: %v", err)
				}

				fields := []struct {
					path string
					cmp  func(a, b interface{}) bool
				}{
					{path: "import_hash", cmp: func(a, b interface{}) bool { return fmt.Sprint(a) == fmt.Sprint(b) }},
					{path: "imphash", cmp: func(a, b interface{}) bool { return fmt.Sprint(a) == fmt.Sprint(b) }},
					{path: "symhash", cmp: func(a, b interface{}) bool { return fmt.Sprint(a) == fmt.Sprint(b) }},
					{path: "imports", cmp: approxImports(format, builder)},
					{path: "imports_names_entropy", cmp: approxFloat64(0.1)},
					{path: "imports_names_var_entropy", cmp: approxFloat64(0.01)},
					{path: "go_import_hash", cmp: approxHash(format, builder)},
					{path: "go_imports", cmp: approxImports(format, builder)},
					{path: "go_imports_names_entropy", cmp: approxFloat64(0.1)},
					{path: "go_imports_names_var_entropy", cmp: approxFloat64(0.01)},
					{path: "go_stripped", cmp: func(a, b interface{}) bool { return a == b }},
					{path: "sections", cmp: approxSections(0.1)},
				}

				for _, f := range fields {
					path := format + "." + f.path
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
}

func approxHash(format, builder string) func(a, b interface{}) bool {
	return func(a, b interface{}) bool {
		as := fmt.Sprint(a)
		bs := fmt.Sprint(b)
		if len(as) != len(bs) {
			return false
		}
		if format == "macho" && builder == "garble" {
			// We can't know more since the hash depends on Go version.
			return true
		}
		return as == bs
	}
}

func approxImports(format, builder string) func(a, b interface{}) bool {
	return func(a, b interface{}) bool {
		as, ok := a.([]string)
		if !ok {
			return false
		}
		bs, ok := b.([]string)
		if !ok {
			return false
		}
		if format == "macho" && builder == "garble" {
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
			if ((aObj[i].VarEntropy == nil) != (bObj[i].VarEntropy == nil)) || (aObj[i].VarEntropy != nil && math.Abs(*aObj[i].VarEntropy-*bObj[i].VarEntropy) > tol) {
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
	varEntropy := "<nil>"
	if o.VarEntropy != nil {
		varEntropy = strconv.FormatFloat(*o.VarEntropy, 'f', -1, 64)
	}
	return fmt.Sprintf("{Name: %q, Size: %s, Entropy: %s, VarEntropy: %s}", name, size, entropy, varEntropy)
}

var want = map[string]mapstr.M{
	"go_pe": {
		"pe": mapstr.M{
			"imphash":                      "c7269d59926fa4252270f407e4dab043",
			"go_import_hash":               "10bddcb4cee42080f76c88d9ff964491",
			"go_imports_names_entropy":     4.156563879566413,
			"go_imports_names_var_entropy": 0.0014785066641319837,
			"go_stripped":                  false,
			"sections": []objSection{
				{Name: strPtr(".text"), Size: uint64Ptr(0x8e400), Entropy: float64Ptr(6.17), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".rdata"), Size: uint64Ptr(0x9ea00), Entropy: float64Ptr(5.13), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".data"), Size: uint64Ptr(0x17a00), Entropy: float64Ptr(4.60), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".zdebug_abbrev"), Size: uint64Ptr(0x200), Entropy: float64Ptr(4.82), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".zdebug_line"), Size: uint64Ptr(0x1cc00), Entropy: float64Ptr(7.99), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".zdebug_frame"), Size: uint64Ptr(0x5800), Entropy: float64Ptr(7.92), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".debug_gdb_scripts"), Size: uint64Ptr(0x200), Entropy: float64Ptr(0.84), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".zdebug_info"), Size: uint64Ptr(0x32a00), Entropy: float64Ptr(7.99), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".zdebug_loc"), Size: uint64Ptr(0x1ba00), Entropy: float64Ptr(7.98), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".zdebug_ranges"), Size: uint64Ptr(0x9600), Entropy: float64Ptr(7.78), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".idata"), Size: uint64Ptr(0x600), Entropy: float64Ptr(3.61), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".reloc"), Size: uint64Ptr(0x6a00), Entropy: float64Ptr(5.44), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".symtab"), Size: uint64Ptr(0x17a00), Entropy: float64Ptr(5.12), VarEntropy: float64Ptr(0.0001)},
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
			"imports_names_entropy":     4.2079021689106195,
			"imports_names_var_entropy": 0.0014785066641319837,
			"go_imports": []string{
				"github.com/elastic/beats/v7/auditbeat/module/file_integrity/testdata/b.Used",
				"github.com/elastic/beats/v7/auditbeat/module/file_integrity/testdata/b.hash",
			},
		},
	},
	"go_elf": {
		"elf": mapstr.M{
			"go_imports_names_entropy":     4.156563879566413,
			"go_imports_names_var_entropy": 0.0073028693197579415,
			"go_stripped":                  false,
			"sections": []objSection{
				{Name: strPtr(""), Size: uint64Ptr(0x0), Entropy: float64Ptr(0.0), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".text"), Size: uint64Ptr(0x7ffd6), Entropy: float64Ptr(6.17), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".rodata"), Size: uint64Ptr(0x35940), Entropy: float64Ptr(4.35), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".shstrtab"), Size: uint64Ptr(0x17a), Entropy: float64Ptr(4.33), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".typelink"), Size: uint64Ptr(0x4f0), Entropy: float64Ptr(3.77), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".itablink"), Size: uint64Ptr(0x60), Entropy: float64Ptr(2.14), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".gosymtab"), Size: uint64Ptr(0x0), Entropy: float64Ptr(0.0), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".gopclntab"), Size: uint64Ptr(0x5a5c8), Entropy: float64Ptr(5.48), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".go.buildinfo"), Size: uint64Ptr(0x20), Entropy: float64Ptr(3.56), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".noptrdata"), Size: uint64Ptr(0x10720), Entropy: float64Ptr(5.60), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".data"), Size: uint64Ptr(0x7810), Entropy: float64Ptr(1.60), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".bss"), Size: uint64Ptr(0x2ef48), Entropy: float64Ptr(0), VarEntropy: float64Ptr(0)},
				{Name: strPtr(".noptrbss"), Size: uint64Ptr(0x5360), Entropy: float64Ptr(0), VarEntropy: float64Ptr(0)},
				{Name: strPtr(".zdebug_abbrev"), Size: uint64Ptr(0x119), Entropy: float64Ptr(7.18), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".zdebug_line"), Size: uint64Ptr(0x1b90f), Entropy: float64Ptr(7.99), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".zdebug_frame"), Size: uint64Ptr(0x551b), Entropy: float64Ptr(7.92), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".debug_gdb_scripts"), Size: uint64Ptr(0x31), Entropy: float64Ptr(4.24), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".zdebug_info"), Size: uint64Ptr(0x31a2a), Entropy: float64Ptr(7.99), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".zdebug_loc"), Size: uint64Ptr(0x198d9), Entropy: float64Ptr(7.98), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".zdebug_ranges"), Size: uint64Ptr(0x8fbc), Entropy: float64Ptr(7.78), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".note.go.buildid"), Size: uint64Ptr(0x64), Entropy: float64Ptr(5.38), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".symtab"), Size: uint64Ptr(0xc5e8), Entropy: float64Ptr(3.21), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".strtab"), Size: uint64Ptr(0xb2d6), Entropy: float64Ptr(4.81), VarEntropy: float64Ptr(0.0001)},
			},
			"import_hash":    "d41d8cd98f00b204e9800998ecf8427e",
			"go_import_hash": "10bddcb4cee42080f76c88d9ff964491",
			"go_imports": []string{
				"github.com/elastic/beats/v7/auditbeat/module/file_integrity/testdata/b.Used",
				"github.com/elastic/beats/v7/auditbeat/module/file_integrity/testdata/b.hash",
			},
		},
	},
	"garble_elf": {
		"elf": mapstr.M{
			"import_hash":    "d41d8cd98f00b204e9800998ecf8427e",
			"go_import_hash": "d41d8cd98f00b204e9800998ecf8427e",
			"go_stripped":    true,
			"sections": []objSection{
				{Name: strPtr(""), Size: uint64Ptr(0x0), Entropy: float64Ptr(0.0), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".text"), Size: uint64Ptr(0x74f85), Entropy: float64Ptr(6.18), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".rodata"), Size: uint64Ptr(0x331e4), Entropy: float64Ptr(4.25), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".shstrtab"), Size: uint64Ptr(0x94), Entropy: float64Ptr(4.27), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".typelink"), Size: uint64Ptr(0x4ec), Entropy: float64Ptr(3.69), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".itablink"), Size: uint64Ptr(0x60), Entropy: float64Ptr(2.14), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".gosymtab"), Size: uint64Ptr(0x0), Entropy: float64Ptr(0.0), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".gopclntab"), Size: uint64Ptr(0x56370), Entropy: float64Ptr(5.42), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".go.buildinfo"), Size: uint64Ptr(0x20), Entropy: float64Ptr(3.56), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".noptrdata"), Size: uint64Ptr(0x10720), Entropy: float64Ptr(5.60), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".data"), Size: uint64Ptr(0x7570), Entropy: float64Ptr(1.54), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".bss"), Size: uint64Ptr(0x2ef48), Entropy: float64Ptr(0.0), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr(".noptrbss"), Size: uint64Ptr(0x5340), Entropy: float64Ptr(0.0), VarEntropy: float64Ptr(0.0001)},
			},
		},
	},
	"go_macho": {
		"macho": mapstr.M{
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
				{Name: strPtr("__text"), Size: uint64Ptr(0x8be36), Entropy: float64Ptr(6.16), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__symbol_stub1"), Size: uint64Ptr(0x102), Entropy: float64Ptr(3.62), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__rodata"), Size: uint64Ptr(0x38b4f), Entropy: float64Ptr(4.37), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__typelink"), Size: uint64Ptr(0x550), Entropy: float64Ptr(3.64), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__itablink"), Size: uint64Ptr(0x78), Entropy: float64Ptr(2.63), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__gosymtab"), Size: uint64Ptr(0x0), Entropy: float64Ptr(0.0), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__gopclntab"), Size: uint64Ptr(0x614a0), Entropy: float64Ptr(5.46), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__go_buildinfo"), Size: uint64Ptr(0x20), Entropy: float64Ptr(3.79), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__nl_symbol_ptr"), Size: uint64Ptr(0x158), Entropy: float64Ptr(0.0), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__noptrdata"), Size: uint64Ptr(0x10780), Entropy: float64Ptr(5.59), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__data"), Size: uint64Ptr(0x7470), Entropy: float64Ptr(1.74), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__bss"), Size: uint64Ptr(0x2f068), Entropy: float64Ptr(6.13), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__noptrbss"), Size: uint64Ptr(0x51c0), Entropy: float64Ptr(5.65), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__zdebug_abbrev"), Size: uint64Ptr(0x117), Entropy: float64Ptr(7.16), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__zdebug_line"), Size: uint64Ptr(0x1d615), Entropy: float64Ptr(7.99), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__zdebug_frame"), Size: uint64Ptr(0x5b82), Entropy: float64Ptr(7.92), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__debug_gdb_scri"), Size: uint64Ptr(0x31), Entropy: float64Ptr(4.24), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__zdebug_info"), Size: uint64Ptr(0x33a7b), Entropy: float64Ptr(7.99), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__zdebug_loc"), Size: uint64Ptr(0x1a57f), Entropy: float64Ptr(7.98), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__zdebug_ranges"), Size: uint64Ptr(0x8371), Entropy: float64Ptr(7.89), VarEntropy: float64Ptr(0.0001)},
			},
			"import_hash":                  "d3ccf195b62a9279c3c19af1080497ec",
			"imports_names_entropy":        4.132925542571368,
			"imports_names_var_entropy":    0.002702653338037826,
			"go_import_hash":               "10bddcb4cee42080f76c88d9ff964491",
			"go_imports_names_entropy":     4.156563879566413,
			"go_imports_names_var_entropy": 0.0073028693197579415,
			"go_stripped":                  false,
		},
	},
	"garble_macho": {
		"macho": mapstr.M{
			"sections": []objSection{
				{Name: strPtr("__text"), Size: uint64Ptr(0x80e52), Entropy: float64Ptr(6.17), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__symbol_stub1"), Size: uint64Ptr(0x102), Entropy: float64Ptr(3.62), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__rodata"), Size: uint64Ptr(0x367b3), Entropy: float64Ptr(4.28), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__typelink"), Size: uint64Ptr(0x554), Entropy: float64Ptr(3.85), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__itablink"), Size: uint64Ptr(0x78), Entropy: float64Ptr(2.61), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__gosymtab"), Size: uint64Ptr(0x0), Entropy: float64Ptr(0.0), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__gopclntab"), Size: uint64Ptr(0x5cf68), Entropy: float64Ptr(5.41), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__go_buildinfo"), Size: uint64Ptr(0x20), Entropy: float64Ptr(3.85), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__nl_symbol_ptr"), Size: uint64Ptr(0x158), Entropy: float64Ptr(0.0), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__noptrdata"), Size: uint64Ptr(0x10780), Entropy: float64Ptr(5.59), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__data"), Size: uint64Ptr(0x71f0), Entropy: float64Ptr(1.72), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__bss"), Size: uint64Ptr(0x2f088), Entropy: float64Ptr(6.13), VarEntropy: float64Ptr(0.0001)},
				{Name: strPtr("__noptrbss"), Size: uint64Ptr(0x51a0), Entropy: float64Ptr(5.55), VarEntropy: float64Ptr(0.0001)},
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
			"imports_names_entropy":     4.132925542571368,
			"imports_names_var_entropy": 0.002702653338037826,
			"go_imports": []string{
				"evnQ6ZcH.NEfVFrsU",
				"evnQ6ZcH.NEfVFrsU.func1",
				"evnQ6ZcH.obErrEr2",
				"evnQ6ZcH.obErrEr2.func1",
				"evnQ6ZcH.obErrEr2.func1.1",
				"main.main",
				"main.main.func1",
			},
			"symhash":                      "d3ccf195b62a9279c3c19af1080497ec",
			"go_import_hash":               "ea0346ba1d3c7c7e762864b7abd53399",
			"go_imports_names_entropy":     4.527763863520965,
			"go_imports_names_var_entropy": 0.004284997488747353,
			"go_stripped":                  true,
		},
	},
}
