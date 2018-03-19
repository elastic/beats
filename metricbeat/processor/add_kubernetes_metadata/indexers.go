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

	config := map[string]interface{}{
		"lookup_fields": []string{"metricset.host"},
	}
	fieldCfg, err := common.NewConfigFrom(config)
	if err == nil {
		//Add field matcher with field to lookup as metricset.host
		kubernetes.Indexing.AddDefaultMatcherConfig(kubernetes.FieldMatcherName, *fieldCfg)
	}
}
