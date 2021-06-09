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

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/internal/metrics"
	"github.com/elastic/go-structform/gotype"
)

// Memory holds os-specifc memory usage data
// The vast majority of these values are cross-platform
// However, we're wrapping all them for the sake of safety, and for the more variable swap metrics
type Memory struct {
	Total metrics.OptUint `struct:"total,omitempty"`
	Used  UsedMemStats    `struct:"used,omitempty"`

	Free   metrics.OptUint `struct:"free,omitempty"`
	Cached metrics.OptUint `struct:"cached,omitempty"`
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
	Pct   metrics.OptFloat `struct:"pct,omitempty"`
	Bytes metrics.OptUint  `struct:"bytes,omitempty"`
}

// ActualMemoryMetrics wraps the actual.* memory metrics
type ActualMemoryMetrics struct {
	Free metrics.OptUint `struct:"free,omitempty"`
	Used UsedMemStats    `struct:"used,omitempty"`
}

// SwapMetrics wraps swap.* memory metrics
type SwapMetrics struct {
	Total metrics.OptUint `struct:"total,omitempty"`
	Used  UsedMemStats    `struct:"used,omitempty"`
	Free  metrics.OptUint `struct:"free,omitempty"`
}

// Get returns platform-independent memory metrics.
func Get(procfs string) (Memory, error) {
	base, err := get(procfs)
	if err != nil {
		return Memory{}, errors.Wrap(err, "error getting system memory info")
	}

	// Add percentages
	// In theory, `Used` and `Total` are available everywhere, so assume values are good.
	if base.Total.ValueOrZero() != 0 {
		percUsed := float64(base.Used.Bytes.ValueOrZero()) / float64(base.Total.ValueOrZero())
		base.Used.Pct = metrics.NewFloatValue(common.Round(percUsed, common.DefaultDecimalPlacesCount))

		actualPercUsed := float64(base.Actual.Used.Bytes.ValueOrZero()) / float64(base.Total.ValueOrZero())
		base.Actual.Used.Pct = metrics.NewFloatValue(common.Round(actualPercUsed, common.DefaultDecimalPlacesCount))
	}

	if base.Swap.Total.ValueOrZero() != 0 && base.Swap.Used.Bytes.Exists() {
		perc := float64(base.Swap.Used.Bytes.ValueOrZero()) / float64(base.Swap.Total.ValueOrZero())
		base.Swap.Used.Pct = metrics.NewFloatValue(common.Round(perc, common.DefaultDecimalPlacesCount))
	}

	return base, nil
}

// Format returns a formatted MapStr ready to be sent upstream
func (mem Memory) Format() (common.MapStr, error) {
	to := common.MapStr{}
	unfold, err := gotype.NewUnfolder(nil, gotype.Unfolders(
		metrics.UnfoldOptUint,
		metrics.UnfoldOptFloat,
	))
	if err != nil {
		return nil, errors.Wrap(err, "error creating Folder")
	}
	fold, err := gotype.NewIterator(unfold, gotype.Folders(
		metrics.FoldOptFloat,
		metrics.FoldOptUint,
	))
	if err != nil {
		return nil, errors.Wrap(err, "error creating unfolder")
	}

	unfold.SetTarget(&to)
	if err := fold.Fold(mem); err != nil {
		return nil, errors.Wrap(err, "error folding memory structure")
	}

	return to, nil
}
