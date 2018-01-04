package kibana

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/go-ucfg/yaml"
)

type transformer struct {
	entries []kibanaEntry
}
type kibanaEntry struct {
	Kibana struct {
		SourceFilters []string `config:"source_filters"`
	} `config:"kibana"`
}

func newTransformer(entries []kibanaEntry) *transformer {
	return &transformer{entries: entries}
}

func (t *transformer) transform() common.MapStr {
	transformed := common.MapStr{}

	var srcFilters []common.MapStr
	for _, entry := range t.entries {
		for _, sourceFilter := range entry.Kibana.SourceFilters {
			srcFilters = append(srcFilters, common.MapStr{"value": sourceFilter})
		}
	}
	if len(srcFilters) > 0 {
		transformed["sourceFilters"] = srcFilters
	}
	return transformed
}

func loadKibanaEntriesFromYaml(yamlFile string) ([]kibanaEntry, error) {
	entries := []kibanaEntry{}
	cfg, err := yaml.NewConfigWithFile(yamlFile)
	if err != nil {
		return nil, err
	}
	err = cfg.Unpack(&entries)
	if err != nil {
		return nil, err
	}
	return entries, nil
}
