package log_annotations

import "github.com/elastic/beats/libbeat/common"

type config struct {
	Prefix string           `config:"prefix"`
	Config []*common.Config `config:"config"`
}

func defaultConfig() config {
	rawCfg := map[string]interface{}{
		"type": "docker",
		"containers": map[string]interface{}{
			"ids": []string{
				"${data.container.id}",
			},
		},
	}
	cfg, _ := common.NewConfigFrom(rawCfg)
	return config{
		Prefix: "co.elastic.logs",
		Config: []*common.Config{cfg},
	}
}
