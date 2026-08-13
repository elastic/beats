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

	"github.com/elastic/beats/v7/testing/testutils"
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

				testutils.SkipIfFIPSOnly(t, "file parser uses MD5.")

				got := make(mapstr.M)
				err := exeObjParser(nil).Parse(got, target)
				if err != nil {
					t.Fatalf("unexpected error calling exeObjParser.Parse: %v", err)
				}

				fields := []struct {
					path string
					cmp  func(a, b any) bool
				}{
					{path: "import_hash", cmp: func(a, b any) bool { return fmt.Sprint(a) == fmt.Sprint(b) }},
					{path: "imphash", cmp: func(a, b any) bool { return fmt.Sprint(a) == fmt.Sprint(b) }},
					{path: "symhash", cmp: func(a, b any) bool { return fmt.Sprint(a) == fmt.Sprint(b) }},
					{path: "imports", cmp: approxImports(format, builder)},
					{path: "imports_names_entropy", cmp: approxFloat64(0.1)},
					{path: "imports_names_var_entropy", cmp: approxFloat64(0.01)},
					{path: "go_import_hash", cmp: approxHash(format, builder)},
					{path: "go_imports", cmp: approxImports(format, builder)},
					{path: "go_imports_names_entropy", cmp: approxFloat64(0.1)},
					{path: "go_imports_names_var_entropy", cmp: approxFloat64(0.01)},
					{path: "go_stripped", cmp: func(a, b any) bool { return a == b }},
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

func approxHash(format, builder string) func(a, b any) bool {
	return func(a, b any) bool {
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

func approxImports(format, builder string) func(a, b any) bool {
	return func(a, b any) bool {
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

func approxFloat64(tol float64) func(a, b any) bool {
	return func(a, b any) bool {
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

func approxSections(tol float64) func(a, b any) bool {
	return func(a, b any) bool {
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

//go:fix inline
func strPtr(s string) *string { return new(s) }

//go:fix inline
func float64Ptr(f float64) *float64 { return new(f) }

//go:fix inline
func uint64Ptr(u uint64) *uint64 { return new(u) }

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
				{Name: new(".text"), Size: new(uint64(0x8e400)), Entropy: new(6.17), VarEntropy: new(0.0001)},
				{Name: new(".rdata"), Size: new(uint64(0x9ea00)), Entropy: new(5.13), VarEntropy: new(0.0001)},
				{Name: new(".data"), Size: new(uint64(0x17a00)), Entropy: new(4.60), VarEntropy: new(0.0001)},
				{Name: new(".zdebug_abbrev"), Size: new(uint64(0x200)), Entropy: new(4.82), VarEntropy: new(0.0001)},
				{Name: new(".zdebug_line"), Size: new(uint64(0x1cc00)), Entropy: new(7.99), VarEntropy: new(0.0001)},
				{Name: new(".zdebug_frame"), Size: new(uint64(0x5800)), Entropy: new(7.92), VarEntropy: new(0.0001)},
				{Name: new(".debug_gdb_scripts"), Size: new(uint64(0x200)), Entropy: new(0.84), VarEntropy: new(0.0001)},
				{Name: new(".zdebug_info"), Size: new(uint64(0x32a00)), Entropy: new(7.99), VarEntropy: new(0.0001)},
				{Name: new(".zdebug_loc"), Size: new(uint64(0x1ba00)), Entropy: new(7.98), VarEntropy: new(0.0001)},
				{Name: new(".zdebug_ranges"), Size: new(uint64(0x9600)), Entropy: new(7.78), VarEntropy: new(0.0001)},
				{Name: new(".idata"), Size: new(uint64(0x600)), Entropy: new(3.61), VarEntropy: new(0.0001)},
				{Name: new(".reloc"), Size: new(uint64(0x6a00)), Entropy: new(5.44), VarEntropy: new(0.0001)},
				{Name: new(".symtab"), Size: new(uint64(0x17a00)), Entropy: new(5.12), VarEntropy: new(0.0001)},
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
				{Name: new(""), Size: new(uint64(0x0)), Entropy: new(0.0), VarEntropy: new(0.0)},
				{Name: new(".text"), Size: new(uint64(0x7ffd6)), Entropy: new(6.17), VarEntropy: new(0.0001)},
				{Name: new(".rodata"), Size: new(uint64(0x35940)), Entropy: new(4.36), VarEntropy: new(0.0005)},
				{Name: new(".shstrtab"), Size: new(uint64(0x17a)), Entropy: new(4.33), VarEntropy: new(0.0019)},
				{Name: new(".typelink"), Size: new(uint64(0x4f0)), Entropy: new(3.77), VarEntropy: new(0.0083)},
				{Name: new(".itablink"), Size: new(uint64(0x60)), Entropy: new(2.15), VarEntropy: new(0.046)},
				{Name: new(".gosymtab"), Size: new(uint64(0x0)), Entropy: new(0.0), VarEntropy: new(0.0)},
				{Name: new(".gopclntab"), Size: new(uint64(0x5a5c8)), Entropy: new(5.49), VarEntropy: new(0.0001)},
				{Name: new(".go.buildinfo"), Size: new(uint64(0x20)), Entropy: new(3.56), VarEntropy: new(0.07)},
				{Name: new(".noptrdata"), Size: new(uint64(0x10720)), Entropy: new(5.61), VarEntropy: new(0.0001)},
				{Name: new(".data"), Size: new(uint64(0x7810)), Entropy: new(1.60), VarEntropy: new(0.0004)},
				{Name: new(".bss"), Size: new(uint64(0x2ef48)), Entropy: new(0.0), VarEntropy: new(0.0)},
				{Name: new(".noptrbss"), Size: new(uint64(0x5360)), Entropy: new(0.0), VarEntropy: new(0.0)},
				{Name: new(".zdebug_abbrev"), Size: new(uint64(0x1e6)), Entropy: new(4.71), VarEntropy: new(0.007)},
				{Name: new(".zdebug_line"), Size: new(uint64(0x30b61)), Entropy: new(5.93), VarEntropy: new(0.0003)},
				{Name: new(".zdebug_frame"), Size: new(uint64(0xee0c)), Entropy: new(3.59), VarEntropy: new(0.0002)},
				{Name: new(".debug_gdb_scripts"), Size: new(uint64(0x31)), Entropy: new(4.25), VarEntropy: new(0.016)},
				{Name: new(".zdebug_info"), Size: new(uint64(0x79ef9)), Entropy: new(5.80), VarEntropy: new(0.0002)},
				{Name: new(".zdebug_loc"), Size: new(uint64(0x919d5)), Entropy: new(2.62), VarEntropy: new(0.0002)},
				{Name: new(".zdebug_ranges"), Size: new(uint64(0x313b0)), Entropy: new(2.20), VarEntropy: new(0.0006)},
				{Name: new(".note.go.buildid"), Size: new(uint64(0x64)), Entropy: new(5.29), VarEntropy: new(0.012)},
				{Name: new(".symtab"), Size: new(uint64(0xc5e8)), Entropy: new(3.21), VarEntropy: new(0.0003)},
				{Name: new(".strtab"), Size: new(uint64(0xb2d6)), Entropy: new(4.81), VarEntropy: new(0.0004)},
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
				{Name: new(""), Size: new(uint64(0x0)), Entropy: new(0.0), VarEntropy: new(0.0001)},
				{Name: new(".text"), Size: new(uint64(0x74f85)), Entropy: new(6.18), VarEntropy: new(0.0001)},
				{Name: new(".rodata"), Size: new(uint64(0x331e4)), Entropy: new(4.25), VarEntropy: new(0.0001)},
				{Name: new(".shstrtab"), Size: new(uint64(0x94)), Entropy: new(4.27), VarEntropy: new(0.0001)},
				{Name: new(".typelink"), Size: new(uint64(0x4ec)), Entropy: new(3.69), VarEntropy: new(0.0001)},
				{Name: new(".itablink"), Size: new(uint64(0x60)), Entropy: new(2.14), VarEntropy: new(0.0001)},
				{Name: new(".gosymtab"), Size: new(uint64(0x0)), Entropy: new(0.0), VarEntropy: new(0.0001)},
				{Name: new(".gopclntab"), Size: new(uint64(0x56370)), Entropy: new(5.42), VarEntropy: new(0.0001)},
				{Name: new(".go.buildinfo"), Size: new(uint64(0x20)), Entropy: new(3.56), VarEntropy: new(0.0001)},
				{Name: new(".noptrdata"), Size: new(uint64(0x10720)), Entropy: new(5.60), VarEntropy: new(0.0001)},
				{Name: new(".data"), Size: new(uint64(0x7570)), Entropy: new(1.54), VarEntropy: new(0.0001)},
				{Name: new(".bss"), Size: new(uint64(0x2ef48)), Entropy: new(0.0), VarEntropy: new(0.0001)},
				{Name: new(".noptrbss"), Size: new(uint64(0x5340)), Entropy: new(0.0), VarEntropy: new(0.0001)},
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
				{Name: new("__text"), Size: new(uint64(0x8be36)), Entropy: new(6.16), VarEntropy: new(0.0001)},
				{Name: new("__symbol_stub1"), Size: new(uint64(0x102)), Entropy: new(3.62), VarEntropy: new(0.0001)},
				{Name: new("__rodata"), Size: new(uint64(0x38b4f)), Entropy: new(4.37), VarEntropy: new(0.0001)},
				{Name: new("__typelink"), Size: new(uint64(0x550)), Entropy: new(3.64), VarEntropy: new(0.0001)},
				{Name: new("__itablink"), Size: new(uint64(0x78)), Entropy: new(2.63), VarEntropy: new(0.0001)},
				{Name: new("__gosymtab"), Size: new(uint64(0x0)), Entropy: new(0.0), VarEntropy: new(0.0001)},
				{Name: new("__gopclntab"), Size: new(uint64(0x614a0)), Entropy: new(5.46), VarEntropy: new(0.0001)},
				{Name: new("__go_buildinfo"), Size: new(uint64(0x20)), Entropy: new(3.79), VarEntropy: new(0.0001)},
				{Name: new("__nl_symbol_ptr"), Size: new(uint64(0x158)), Entropy: new(0.0), VarEntropy: new(0.0001)},
				{Name: new("__noptrdata"), Size: new(uint64(0x10780)), Entropy: new(5.59), VarEntropy: new(0.0001)},
				{Name: new("__data"), Size: new(uint64(0x7470)), Entropy: new(1.74), VarEntropy: new(0.0001)},
				{Name: new("__bss"), Size: new(uint64(0x2f068)), Entropy: new(6.13), VarEntropy: new(0.0001)},
				{Name: new("__noptrbss"), Size: new(uint64(0x51c0)), Entropy: new(5.65), VarEntropy: new(0.0001)},
				{Name: new("__zdebug_abbrev"), Size: new(uint64(0x117)), Entropy: new(7.16), VarEntropy: new(0.0001)},
				{Name: new("__zdebug_line"), Size: new(uint64(0x1d615)), Entropy: new(7.99), VarEntropy: new(0.0001)},
				{Name: new("__zdebug_frame"), Size: new(uint64(0x5b82)), Entropy: new(7.92), VarEntropy: new(0.0001)},
				{Name: new("__debug_gdb_scri"), Size: new(uint64(0x31)), Entropy: new(4.24), VarEntropy: new(0.0001)},
				{Name: new("__zdebug_info"), Size: new(uint64(0x33a7b)), Entropy: new(7.99), VarEntropy: new(0.0001)},
				{Name: new("__zdebug_loc"), Size: new(uint64(0x1a57f)), Entropy: new(7.98), VarEntropy: new(0.0001)},
				{Name: new("__zdebug_ranges"), Size: new(uint64(0x8371)), Entropy: new(7.89), VarEntropy: new(0.0001)},
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
				{Name: new("__text"), Size: new(uint64(0x80e52)), Entropy: new(6.17), VarEntropy: new(0.0001)},
				{Name: new("__symbol_stub1"), Size: new(uint64(0x102)), Entropy: new(3.62), VarEntropy: new(0.0001)},
				{Name: new("__rodata"), Size: new(uint64(0x367b3)), Entropy: new(4.28), VarEntropy: new(0.0001)},
				{Name: new("__typelink"), Size: new(uint64(0x554)), Entropy: new(3.85), VarEntropy: new(0.0001)},
				{Name: new("__itablink"), Size: new(uint64(0x78)), Entropy: new(2.61), VarEntropy: new(0.0001)},
				{Name: new("__gosymtab"), Size: new(uint64(0x0)), Entropy: new(0.0), VarEntropy: new(0.0001)},
				{Name: new("__gopclntab"), Size: new(uint64(0x5cf68)), Entropy: new(5.41), VarEntropy: new(0.0001)},
				{Name: new("__go_buildinfo"), Size: new(uint64(0x20)), Entropy: new(3.85), VarEntropy: new(0.0001)},
				{Name: new("__nl_symbol_ptr"), Size: new(uint64(0x158)), Entropy: new(0.0), VarEntropy: new(0.0001)},
				{Name: new("__noptrdata"), Size: new(uint64(0x10780)), Entropy: new(5.59), VarEntropy: new(0.0001)},
				{Name: new("__data"), Size: new(uint64(0x71f0)), Entropy: new(1.72), VarEntropy: new(0.0001)},
				{Name: new("__bss"), Size: new(uint64(0x2f088)), Entropy: new(6.13), VarEntropy: new(0.0001)},
				{Name: new("__noptrbss"), Size: new(uint64(0x51a0)), Entropy: new(5.55), VarEntropy: new(0.0001)},
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
