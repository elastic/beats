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

package memory

import (
	"fmt"

	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/elastic-agent-system-metrics/metric"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

// Memory holds os-specifc memory usage data
// The vast majority of these values are cross-platform
// However, we're wrapping all them for the sake of safety, and for the more variable swap metrics
type Memory struct {
	Total opt.Uint     `struct:"total,omitempty"`
	Used  UsedMemStats `struct:"used,omitempty"`

	Free   opt.Uint `struct:"free,omitempty"`
	Cached opt.Uint `struct:"cached,omitempty"`
	// "Actual" values are, technically, a linux-only concept
	// For better or worse we've expanded it to include "derived"
	// Memory values on other platforms, which we should
	// probably keep for the sake of backwards compatibility
	// However, because the derived value varies from platform to platform,
	// We may want to more precisely document what these mean.
	Actual ActualMemoryMetrics `struct:"actual,omitempty"`

	// Swap metrics
	Swap SwapMetrics `struct:"swap,omitempty"`

	// Zswap metrics (Linux only, available when zswap is enabled)
	Zswap ZswapMetrics `struct:"zswap,omitempty"`
}

// UsedMemStats wraps used.* memory metrics
type UsedMemStats struct {
	Pct   opt.Float `struct:"pct,omitempty"`
	Bytes opt.Uint  `struct:"bytes,omitempty"`
}

// ActualMemoryMetrics wraps the actual.* memory metrics
type ActualMemoryMetrics struct {
	Free opt.Uint     `struct:"free,omitempty"`
	Used UsedMemStats `struct:"used,omitempty"`
}

// SwapMetrics wraps swap.* memory metrics
type SwapMetrics struct {
	Total opt.Uint     `struct:"total,omitempty"`
	Used  UsedMemStats `struct:"used,omitempty"`
	Free  opt.Uint     `struct:"free,omitempty"`
}

// ZswapMetrics wraps zswap.* memory metrics
type ZswapMetrics struct {
	// Compressed is the current compressed size of data in zswap (bytes) from /proc/meminfo
	Compressed opt.Uint `struct:"compressed,omitempty"`
	// Uncompressed is the original uncompressed size of data in zswap (bytes) from /proc/meminfo
	Uncompressed opt.Uint `struct:"uncompressed,omitempty"`
	// Debug contains optional detailed statistics from /sys/kernel/debug/zswap (requires debugfs access)
	Debug ZswapDebugMetrics `struct:"debug,omitempty"`
}

// ZswapDebugMetrics contains detailed zswap statistics from /sys/kernel/debug/zswap.
// These metrics are optional and require debugfs to be mounted and accessible (typically requires root).
type ZswapDebugMetrics struct {
	// PoolLimitHit is the number of times the pool limit was reached
	PoolLimitHit opt.Uint `struct:"pool_limit_hit,omitempty"`
	// PoolTotalSize is the total size of the zswap pool in bytes
	PoolTotalSize opt.Uint `struct:"pool_total_size,omitempty"`
	// RejectAllocFail is the number of pages rejected due to zpool allocation failure
	RejectAllocFail opt.Uint `struct:"reject_alloc_fail,omitempty"`
	// RejectCompressFail is the number of pages rejected due to compression failure
	RejectCompressFail opt.Uint `struct:"reject_compress_fail,omitempty"`
	// RejectCompressPoor is the number of pages rejected due to poor compression ratio
	RejectCompressPoor opt.Uint `struct:"reject_compress_poor,omitempty"`
	// RejectKmemcacheFail is the number of pages rejected due to kmemcache allocation failure
	RejectKmemcacheFail opt.Uint `struct:"reject_kmemcache_fail,omitempty"`
	// RejectReclaimFail is the number of pages rejected due to reclaim failure
	RejectReclaimFail opt.Uint `struct:"reject_reclaim_fail,omitempty"`
	// StoredPages is the number of pages currently stored in zswap
	StoredPages opt.Uint `struct:"stored_pages,omitempty"`
	// WrittenBackPages is the number of pages written back from zswap to swap
	WrittenBackPages opt.Uint `struct:"written_back_pages,omitempty"`
}

// IsZero implements the zeroer interface for structform's folders
func (zswap ZswapMetrics) IsZero() bool {
	return zswap.Compressed.IsZero() && zswap.Uncompressed.IsZero() && zswap.Debug.IsZero()
}

// IsZero implements the zeroer interface for structform's folders
func (z ZswapDebugMetrics) IsZero() bool {
	return z.StoredPages.IsZero() &&
		z.PoolTotalSize.IsZero() &&
		z.WrittenBackPages.IsZero() &&
		z.RejectCompressPoor.IsZero() &&
		z.RejectCompressFail.IsZero() &&
		z.RejectKmemcacheFail.IsZero() &&
		z.RejectAllocFail.IsZero() &&
		z.RejectReclaimFail.IsZero() &&
		z.PoolLimitHit.IsZero()
}

// Get returns platform-independent memory metrics.
func Get(hostfs resolve.Resolver) (Memory, error) {
	base, err := get(hostfs)
	if err != nil {
		return Memory{}, fmt.Errorf("error getting system memory info: %w", err)
	}
	base.fillPercentages()
	base.Zswap.Debug = getZswapDebugMetrics(hostfs)
	return base, nil
}

// IsZero implements the zeroer interface for structform's folders
func (used UsedMemStats) IsZero() bool {
	return used.Pct.IsZero() && used.Bytes.IsZero()
}

// IsZero implements the zeroer interface for structform's folders
func (swap SwapMetrics) IsZero() bool {
	return swap.Free.IsZero() && swap.Used.IsZero() && swap.Total.IsZero()
}

func (base *Memory) fillPercentages() {
	// Add percentages
	// In theory, `Used` and `Total` are available everywhere, so assume values are good.
	if base.Total.Exists() && base.Total.ValueOr(0) != 0 {
		percUsed := float64(base.Used.Bytes.ValueOr(0)) / float64(base.Total.ValueOr(1))
		base.Used.Pct = opt.FloatWith(metric.Round(percUsed))

		actualPercUsed := float64(base.Actual.Used.Bytes.ValueOr(0)) / float64(base.Total.ValueOr(0))
		base.Actual.Used.Pct = opt.FloatWith(metric.Round(actualPercUsed))
	}

	if base.Swap.Total.ValueOr(0) != 0 && base.Swap.Used.Bytes.Exists() {
		perc := float64(base.Swap.Used.Bytes.ValueOr(0)) / float64(base.Swap.Total.ValueOr(0))
		base.Swap.Used.Pct = opt.FloatWith(metric.Round(perc))
	}
}
