package add_kubernetes_metadata

import (
	"github.com/elastic/beats/libbeat/common"
	kubernetes "github.com/elastic/beats/libbeat/processors/add_kubernetes_metadata"
)

func init() {
	// Register default indexers
	cfg := common.NewConfig()

	//Add IP Port Indexer as a default indexer
	kubernetes.Indexing.AddDefaultIndexerConfig(kubernetes.IPPortIndexerName, *cfg)

	formatCfg, err := common.NewConfigFrom(map[string]interface{}{
		"format": "%{[ip]}:%{[port]}",
	})
	if err == nil {
		//Add field matcher with field to lookup as metricset.host
		kubernetes.Indexing.AddDefaultMatcherConfig(kubernetes.FieldFormatMatcherName, *formatCfg)
	}
}
