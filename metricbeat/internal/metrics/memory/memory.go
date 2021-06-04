package memory

import (
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/internal/metrics"
	"github.com/pkg/errors"
)

// Memory holds os-specifc memory usage data
// The vast majority of these values are cross-platform
// However, we're wrapping all them for the sake of safety, and for the more variable swap metrics
type Memory struct {
	Total  metrics.OptUint
	Used   metrics.OptUint
	Free   metrics.OptUint
	Cached metrics.OptUint
	// Actual values are, technically, a linux-only concept
	// For better or worse we've expanded it to include "derived"
	// Memory values on other platforms, which we should
	// probably keep for the sake of backwards compatibility
	ActualFree metrics.OptUint
	ActualUsed metrics.OptUint

	// Derived values
	UsedPercent       metrics.OptFloat
	UsedActualPercent metrics.OptFloat

	// Swap metrics
	SwapTotal       metrics.OptUint
	SwapUsed        metrics.OptUint
	SwapFree        metrics.OptUint
	SwapUsedPercent metrics.OptFloat
}

func newMemory() Memory {
	return Memory{
		Total:             metrics.NewUint(),
		Used:              metrics.NewUint(),
		Free:              metrics.NewUint(),
		Cached:            metrics.NewUint(),
		ActualFree:        metrics.NewUint(),
		ActualUsed:        metrics.NewUint(),
		UsedPercent:       metrics.NewFloat(),
		UsedActualPercent: metrics.NewFloat(),

		SwapTotal:       metrics.NewUint(),
		SwapUsed:        metrics.NewUint(),
		SwapFree:        metrics.NewUint(),
		SwapUsedPercent: metrics.NewFloat(),
	}
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
		percUsed := float64(base.Used.ValueOrZero()) / float64(base.Total.ValueOrZero())
		base.UsedPercent.Some(common.Round(percUsed, common.DefaultDecimalPlacesCount))

		actualPercUsed := float64(base.ActualUsed.ValueOrZero()) / float64(base.Total.ValueOrZero())
		base.UsedActualPercent.Some(common.Round(actualPercUsed, common.DefaultDecimalPlacesCount))
	}

	if base.SwapTotal.ValueOrZero() != 0 && base.SwapUsed.Exists() {
		perc := float64(base.SwapUsed.ValueOrZero()) / float64(base.SwapTotal.ValueOrZero())
		base.SwapUsedPercent.Some(common.Round(perc, common.DefaultDecimalPlacesCount))
	}

	return base, nil
}
