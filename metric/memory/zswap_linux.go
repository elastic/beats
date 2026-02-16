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

package memory

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

const zswapDebugPath = "/sys/kernel/debug/zswap"

// getZswapDebugMetrics reads zswap debug metrics from /sys/kernel/debug/zswap.
// This function returns an empty struct (not an error) if the debug path is not accessible,
// as these metrics are optional and require debugfs to be mounted with appropriate permissions.
func getZswapDebugMetrics(rootfs resolve.Resolver) ZswapDebugMetrics {
	basePath := rootfs.ResolveHostFS(zswapDebugPath)

	// Check if the debug path exists and is accessible
	if _, err := os.Stat(basePath); err != nil {
		return ZswapDebugMetrics{}
	}

	// Read each metric file, ignoring errors for individual files
	return ZswapDebugMetrics{
		PoolLimitHit:        readUintFromFile(filepath.Join(basePath, "pool_limit_hit")),
		PoolTotalSize:       readUintFromFile(filepath.Join(basePath, "pool_total_size")),
		RejectAllocFail:     readUintFromFile(filepath.Join(basePath, "reject_alloc_fail")),
		RejectCompressFail:  readUintFromFile(filepath.Join(basePath, "reject_compress_fail")),
		RejectCompressPoor:  readUintFromFile(filepath.Join(basePath, "reject_compress_poor")),
		RejectKmemcacheFail: readUintFromFile(filepath.Join(basePath, "reject_kmemcache_fail")),
		RejectReclaimFail:   readUintFromFile(filepath.Join(basePath, "reject_reclaim_fail")),
		StoredPages:         readUintFromFile(filepath.Join(basePath, "stored_pages")),
		WrittenBackPages:    readUintFromFile(filepath.Join(basePath, "written_back_pages")),
	}
}

// readUintFromFile reads a uint64 value from a file, returning an empty opt.Uint on any error.
func readUintFromFile(path string) opt.Uint {
	data, err := os.ReadFile(path)
	if err != nil {
		return opt.NewUintNone()
	}

	value, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return opt.NewUintNone()
	}

	return opt.UintWith(value)
}
