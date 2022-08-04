package cmd

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/x-pack/libbeat/management"
)

//fFormatMetricbeatModules is a combination of the map and rename rules in the metricbeat spec file,
// and formats various key values needed by metricbeat
func formatMetricbeatModules(rawIn *management.UnitsConfig) {
	// Extract the module name from the type, usually in the form system/metric
	module := strings.Split(rawIn.UnitType, "/")[0]

	for iter := range rawIn.Streams {
		rawIn.Streams[iter]["module"] = module
	}

}

func metricbeatCfg(rawIn management.UnitsConfig) ([]*reload.ConfigWithMeta, error) {
	management.InjectStreamProcessor(&rawIn, "metrics")
	management.InjectIndexProcessor(&rawIn, "metrics")
	formatMetricbeatModules(&rawIn)

	// format for the reloadable list needed bythe cm.Reload() method
	configList, err := management.CreateReloadConfigFromStreams(rawIn)
	if err != nil {
		return nil, fmt.Errorf("error creating reloader config: %w", err)
	}

	return configList, nil
}
