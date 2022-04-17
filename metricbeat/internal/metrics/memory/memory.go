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
	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/metric/system/resolve"
	"github.com/menderesk/beats/v7/libbeat/opt"
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

// Get returns platform-independent memory metrics.
func Get(procfs resolve.Resolver) (Memory, error) {
	base, err := get(procfs)
	if err != nil {
		return Memory{}, errors.Wrap(err, "error getting system memory info")
	}
	base.fillPercentages()
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
		base.Used.Pct = opt.FloatWith(common.Round(percUsed, common.DefaultDecimalPlacesCount))

		actualPercUsed := float64(base.Actual.Used.Bytes.ValueOr(0)) / float64(base.Total.ValueOr(0))
		base.Actual.Used.Pct = opt.FloatWith(common.Round(actualPercUsed, common.DefaultDecimalPlacesCount))
	}

	if base.Swap.Total.ValueOr(0) != 0 && base.Swap.Used.Bytes.Exists() {
		perc := float64(base.Swap.Used.Bytes.ValueOr(0)) / float64(base.Swap.Total.ValueOr(0))
		base.Swap.Used.Pct = opt.FloatWith(common.Round(perc, common.DefaultDecimalPlacesCount))
	}
}
