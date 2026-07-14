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

package tlscommon

import (
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

// These benchmarks quantify the memory/allocation impact of the
// CertReloader/CAReloader hot-reload machinery (added in #404, #412, #417,
// #419) relative to the static, load-once-and-hold behavior that preceded
// it. They're meant to answer concrete questions raised about that impact:
//
//  1. How much extra memory does enabling reload add per configured TLS
//     endpoint (i.e. per Config/ServerConfig, not per connection)?
//     BenchmarkCertLoad, BenchmarkCAPoolLoad, BenchmarkLoadTLSConfig,
//     BenchmarkLoadTLSServerConfig, BenchmarkTLSConfigMemoryFootprint.
//  2. Does the cost scale with the number of TLS handshakes/connections
//     served by an already-loaded config, or only with the number of
//     distinct configs and the reload interval?
//     BenchmarkCertReloader_GetCertificate_WithinInterval,
//     BenchmarkCAReloader_GetCertPool_WithinInterval,
//     BenchmarkCertReloader_GetCertificate_ReloadEveryCall,
//     BenchmarkCAReloader_GetCertPool_ReloadEveryCall.
//  3. Does memory grow over the life of a single reloader as it actually
//     hot-reloads many times, or does each cycle's old cert/pool get
//     collected? BenchmarkCertReloader_HeapAfterManyReloadCycles,
//     BenchmarkCAReloader_HeapAfterManyReloadCycles.
//  4. Does reload cost scale linearly with the number of independently
//     configured TLS endpoints, or is there cross-endpoint contention/
//     super-linear blowup? BenchmarkReloadCost_ScalingWithEndpointCount.
//
// These benchmarks reuse the writeCAFile/writeKeyAndCertFiles helpers from
// ca_reloader_test.go/cert_reloader_test.go (and, transitively,
// makeKeyCertPair from tlscommon_test.go), which accept testing.TB so they
// work from both *testing.T and *testing.B.

// BenchmarkCertLoad isolates the marginal cost of wrapping a loaded
// certificate in a CertReloader versus the pre-reload behavior of loading it
// once into a bare *tls.Certificate. Both subtests re-read and re-parse the
// same files on every iteration, so the delta between them is the
// CertReloader struct itself (mutex, logger, path strings, timestamp) - not
// the underlying certificate data, which is the same size either way.
func BenchmarkCertLoad(b *testing.B) {
	dir := b.TempDir()
	certPath, keyPath := writeKeyAndCertFiles(b, dir)

	b.Run("StaticNoReload", func(b *testing.B) {
		cfg := &CertificateConfig{Certificate: certPath, Key: keyPath}
		b.ReportAllocs()
		var sink *tls.Certificate
		for i := 0; i < b.N; i++ {
			cert, err := LoadCertificate(cfg, logp.NewNopLogger())
			if err != nil {
				b.Fatal(err)
			}
			sink = cert
		}
		runtime.KeepAlive(sink)
	})

	b.Run("CertReloader", func(b *testing.B) {
		b.ReportAllocs()
		var sink *CertReloader
		for i := 0; i < b.N; i++ {
			r, err := NewCertReloader(certPath, keyPath, logp.NewNopLogger())
			if err != nil {
				b.Fatal(err)
			}
			sink = r
		}
		runtime.KeepAlive(sink)
	})
}

// BenchmarkCAPoolLoad is the CAReloader equivalent of BenchmarkCertLoad.
func BenchmarkCAPoolLoad(b *testing.B) {
	dir := b.TempDir()
	caPaths := []string{writeCAFile(b, dir, "ca.pem")}

	b.Run("StaticNoReload", func(b *testing.B) {
		b.ReportAllocs()
		var sink *staticCertPool
		for i := 0; i < b.N; i++ {
			pool, errs := LoadCertificateAuthorities(caPaths, logp.NewNopLogger())
			if len(errs) > 0 {
				b.Fatal(errs)
			}
			sink = newStaticCertPool(pool)
		}
		runtime.KeepAlive(sink)
	})

	b.Run("CAReloader", func(b *testing.B) {
		b.ReportAllocs()
		var sink *CAReloader
		for i := 0; i < b.N; i++ {
			r, err := NewCAReloader(caPaths, time.Hour, logp.NewNopLogger())
			if err != nil {
				b.Fatal(err)
			}
			sink = r
		}
		runtime.KeepAlive(sink)
	})
}

// BenchmarkLoadTLSConfig measures the end-to-end cost of LoadTLSConfig - the
// function every TLS-enabled client config actually goes through - with
// certificate_reload on versus off. This is the realistic "per configured
// endpoint" cost, combining both the cert and CA paths.
func BenchmarkLoadTLSConfig(b *testing.B) {
	dir := b.TempDir()
	caPath := writeCAFile(b, dir, "ca.pem")
	certPath, keyPath := writeKeyAndCertFiles(b, dir)
	logger := logp.NewNopLogger()

	disabled, enabled := false, true

	b.Run("ReloadDisabled", func(b *testing.B) {
		cfg := &Config{
			CAs:               []string{caPath},
			Certificate:       CertificateConfig{Certificate: certPath, Key: keyPath},
			CertificateReload: CertificateReload{Enabled: &disabled},
		}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := LoadTLSConfig(cfg, logger); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ReloadEnabled", func(b *testing.B) {
		cfg := &Config{
			CAs:               []string{caPath},
			Certificate:       CertificateConfig{Certificate: certPath, Key: keyPath},
			CertificateReload: CertificateReload{Enabled: &enabled, ReloadInterval: time.Hour},
		}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := LoadTLSConfig(cfg, logger); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkLoadTLSServerConfig is the server-side (LoadTLSServerConfig)
// equivalent of BenchmarkLoadTLSConfig.
func BenchmarkLoadTLSServerConfig(b *testing.B) {
	dir := b.TempDir()
	caPath := writeCAFile(b, dir, "ca.pem")
	certPath, keyPath := writeKeyAndCertFiles(b, dir)
	logger := logp.NewNopLogger()

	disabled, enabled := false, true

	b.Run("ReloadDisabled", func(b *testing.B) {
		cfg := &ServerConfig{
			CAs:               []string{caPath},
			Certificate:       CertificateConfig{Certificate: certPath, Key: keyPath},
			CertificateReload: CertificateReload{Enabled: &disabled},
		}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := LoadTLSServerConfig(cfg, logger); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ReloadEnabled", func(b *testing.B) {
		cfg := &ServerConfig{
			CAs:               []string{caPath},
			Certificate:       CertificateConfig{Certificate: certPath, Key: keyPath},
			CertificateReload: CertificateReload{Enabled: &enabled, ReloadInterval: time.Hour},
		}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := LoadTLSServerConfig(cfg, logger); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkCertReloader_GetCertificate_WithinInterval simulates many TLS
// handshakes/connections (b.N of them) served by a single, already-loaded
// CertReloader whose reload interval never elapses. If reloader instances
// "accumulated" with connection volume, allocs/op here would grow with b.N;
// instead it should stay flat (an RLock + pointer read), showing that
// connection/handshake volume does not drive memory growth - only the
// number of distinct configured endpoints does.
func BenchmarkCertReloader_GetCertificate_WithinInterval(b *testing.B) {
	dir := b.TempDir()
	certPath, keyPath := writeKeyAndCertFiles(b, dir)
	r, err := NewCertReloader(certPath, keyPath, logp.NewNopLogger(), WithReloadInterval(time.Hour))
	if err != nil {
		b.Fatalf("creating cert reloader: %v", err)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := r.GetCertificate(nil); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCAReloader_GetCertPool_WithinInterval is the CAReloader
// equivalent of BenchmarkCertReloader_GetCertificate_WithinInterval.
func BenchmarkCAReloader_GetCertPool_WithinInterval(b *testing.B) {
	dir := b.TempDir()
	caPath := writeCAFile(b, dir, "ca.pem")
	r, err := NewCAReloader([]string{caPath}, time.Hour, logp.NewNopLogger())
	if err != nil {
		b.Fatalf("creating CA reloader: %v", err)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = r.GetCertPool()
	}
}

// BenchmarkCertReloader_GetCertificate_ReloadEveryCall forces a real
// disk-read-and-reparse on every call (reload interval effectively zero).
// This is the worst-case steady-state cost of the reload feature: unlike the
// hot-path benchmarks above, it scales with the number of distinct reloader
// instances times how often each one actually reloads (1/reload_interval),
// never with connection/handshake count.
func BenchmarkCertReloader_GetCertificate_ReloadEveryCall(b *testing.B) {
	dir := b.TempDir()
	certPath, keyPath := writeKeyAndCertFiles(b, dir)
	r, err := NewCertReloader(certPath, keyPath, logp.NewNopLogger(), WithReloadInterval(time.Nanosecond))
	if err != nil {
		b.Fatalf("creating cert reloader: %v", err)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := r.GetCertificate(nil); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCAReloader_GetCertPool_ReloadEveryCall is the CAReloader
// equivalent of BenchmarkCertReloader_GetCertificate_ReloadEveryCall.
func BenchmarkCAReloader_GetCertPool_ReloadEveryCall(b *testing.B) {
	dir := b.TempDir()
	caPath := writeCAFile(b, dir, "ca.pem")
	r, err := NewCAReloader([]string{caPath}, time.Nanosecond, logp.NewNopLogger())
	if err != nil {
		b.Fatalf("creating CA reloader: %v", err)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = r.GetCertPool()
	}
}

// BenchmarkTLSConfigMemoryFootprint reports an absolute bytes-per-instance
// figure for holding many distinct, simultaneously-loaded TLS configs (e.g.
// one per output/listener in a fleet of agents) with reload on versus off.
// This is the axis the reload feature actually scales on - number of
// configured endpoints, not number of connections.
//
// Because each b.N iteration internally does n LoadTLSConfig calls, run this
// benchmark with a small explicit -benchtime (e.g. -benchtime=3x) rather than
// the default auto-scaling or a large -benchtime=Nx shared with other
// benchmarks in the same invocation - otherwise the work multiplies to
// b.N*n LoadTLSConfig calls.
func BenchmarkTLSConfigMemoryFootprint(b *testing.B) {
	const n = 1000

	dir := b.TempDir()
	caPath := writeCAFile(b, dir, "ca.pem")
	certPath, keyPath := writeKeyAndCertFiles(b, dir)
	logger := logp.NewNopLogger()
	disabled, enabled := false, true

	run := func(b *testing.B, reload CertificateReload) {
		cfg := &Config{
			CAs:               []string{caPath},
			Certificate:       CertificateConfig{Certificate: certPath, Key: keyPath},
			CertificateReload: reload,
		}

		for i := 0; i < b.N; i++ {
			configs := make([]*TLSConfig, 0, n)

			runtime.GC()
			var before runtime.MemStats
			runtime.ReadMemStats(&before)

			for j := 0; j < n; j++ {
				tlsCfg, err := LoadTLSConfig(cfg, logger)
				if err != nil {
					b.Fatal(err)
				}
				configs = append(configs, tlsCfg)
			}

			runtime.GC()
			var after runtime.MemStats
			runtime.ReadMemStats(&after)

			b.ReportMetric(float64(after.HeapAlloc-before.HeapAlloc)/float64(n), "bytes/config")
			runtime.KeepAlive(configs)
		}
	}

	b.Run("ReloadDisabled", func(b *testing.B) {
		run(b, CertificateReload{Enabled: &disabled})
	})
	b.Run("ReloadEnabled", func(b *testing.B) {
		run(b, CertificateReload{Enabled: &enabled, ReloadInterval: time.Hour})
	})
}

// The *_ReloadEveryCall benchmarks above measure bytes allocated per reload
// (B/op from -benchmem), which is allocation traffic, not retained memory: a
// reloader that leaked its old cert/pool on every cycle instead of freeing it
// would show the exact same B/op, since that metric doesn't distinguish
// garbage from survivors. BenchmarkCertReloader_HeapAfterManyReloadCycles and
// BenchmarkCAReloader_HeapAfterManyReloadCycles close that gap by checking
// retained heap on a single long-lived reloader across many reload cycles: if
// each reload's previous cert/pool were being kept alive instead of collected,
// heap growth would scale with the number of cycles rather than staying flat.

// BenchmarkCertReloader_HeapAfterManyReloadCycles hot-reloads a single
// CertReloader many times (reload interval effectively zero, so every call
// re-reads and re-parses the cert/key from disk) and compares retained heap
// after a short warm-up window against retained heap after many more cycles.
// A flat (near-zero or negative, within GC noise) delta shows old
// certificates are actually being collected each cycle rather than
// accumulating; a delta that scales with cycles would indicate a leak.
func BenchmarkCertReloader_HeapAfterManyReloadCycles(b *testing.B) {
	const warmupCycles = 100
	const measuredCycles = 20000

	dir := b.TempDir()
	certPath, keyPath := writeKeyAndCertFiles(b, dir)

	for i := 0; i < b.N; i++ {
		r, err := NewCertReloader(certPath, keyPath, logp.NewNopLogger(), WithReloadInterval(time.Nanosecond))
		if err != nil {
			b.Fatal(err)
		}

		for c := 0; c < warmupCycles; c++ {
			if _, err := r.GetCertificate(nil); err != nil {
				b.Fatal(err)
			}
		}
		runtime.GC()
		var early runtime.MemStats
		runtime.ReadMemStats(&early)

		for c := 0; c < measuredCycles; c++ {
			if _, err := r.GetCertificate(nil); err != nil {
				b.Fatal(err)
			}
		}
		runtime.GC()
		var late runtime.MemStats
		runtime.ReadMemStats(&late)

		b.ReportMetric(float64(late.HeapAlloc)-float64(early.HeapAlloc), "heap-growth-bytes")
		runtime.KeepAlive(r)
	}
}

// BenchmarkCAReloader_HeapAfterManyReloadCycles is the CAReloader equivalent
// of BenchmarkCertReloader_HeapAfterManyReloadCycles.
func BenchmarkCAReloader_HeapAfterManyReloadCycles(b *testing.B) {
	const warmupCycles = 100
	const measuredCycles = 20000

	dir := b.TempDir()
	caPath := writeCAFile(b, dir, "ca.pem")

	for i := 0; i < b.N; i++ {
		r, err := NewCAReloader([]string{caPath}, time.Nanosecond, logp.NewNopLogger())
		if err != nil {
			b.Fatal(err)
		}

		for c := 0; c < warmupCycles; c++ {
			_ = r.GetCertPool()
		}
		runtime.GC()
		var early runtime.MemStats
		runtime.ReadMemStats(&early)

		for c := 0; c < measuredCycles; c++ {
			_ = r.GetCertPool()
		}
		runtime.GC()
		var late runtime.MemStats
		runtime.ReadMemStats(&late)

		b.ReportMetric(float64(late.HeapAlloc)-float64(early.HeapAlloc), "heap-growth-bytes")
		runtime.KeepAlive(r)
	}
}

// benchEndpoint bundles the reloaders for one independently configured TLS
// endpoint - the unit this feature actually scales with, per the other
// benchmarks in this file (not connections/handshakes).
type benchEndpoint struct {
	cert *CertReloader
	ca   *CAReloader
}

// setupBenchEndpoints creates n independently configured TLS endpoints, each
// with its own cert/key/CA files on disk and its own CertReloader/CAReloader
// (own mutex, own reload timer) so endpoints share no state with each other.
func setupBenchEndpoints(b *testing.B, n int) []benchEndpoint {
	b.Helper()
	root := b.TempDir()
	eps := make([]benchEndpoint, n)
	for i := 0; i < n; i++ {
		dir := filepath.Join(root, fmt.Sprintf("ep%d", i))
		if err := os.MkdirAll(dir, 0o700); err != nil {
			b.Fatalf("creating endpoint dir: %v", err)
		}
		certPath, keyPath := writeKeyAndCertFiles(b, dir)
		caPath := writeCAFile(b, dir, "ca.pem")
		certReloader, err := NewCertReloader(certPath, keyPath, logp.NewNopLogger(), WithReloadInterval(time.Nanosecond))
		if err != nil {
			b.Fatalf("creating cert reloader for endpoint %d: %v", i, err)
		}
		caReloader, err := NewCAReloader([]string{caPath}, time.Nanosecond, logp.NewNopLogger())
		if err != nil {
			b.Fatalf("creating CA reloader for endpoint %d: %v", i, err)
		}
		eps[i] = benchEndpoint{cert: certReloader, ca: caReloader}
	}
	return eps
}

// BenchmarkReloadCost_ScalingWithEndpointCount measures the cost of one
// reload "sweep" - one GetCertificate + one GetCertPool call per configured
// endpoint, each of which reloads from disk since the reload interval is
// effectively zero - as the number of independently configured TLS endpoints
// (N) grows. Endpoints don't share a mutex or any other state, so per-sweep
// cost should scale linearly with N: ns/op and B/op for N=1000 should land
// close to 1000x the N=1 numbers, not super-linearly. That linear
// relationship is the actual scaling axis for this feature's cost - fleet
// wide reload cost is (per-endpoint reload cost) x (number of configured
// endpoints), never a function of connection/handshake volume.
func BenchmarkReloadCost_ScalingWithEndpointCount(b *testing.B) {
	for _, n := range []int{1, 10, 100, 1000} {
		b.Run(fmt.Sprintf("endpoints=%d", n), func(b *testing.B) {
			eps := setupBenchEndpoints(b, n)

			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				for _, ep := range eps {
					if _, err := ep.cert.GetCertificate(nil); err != nil {
						b.Fatal(err)
					}
					_ = ep.ca.GetCertPool()
				}
			}
			b.ReportMetric(float64(n), "endpoints")
		})
	}
}
