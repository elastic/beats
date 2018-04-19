package hints

import "github.com/elastic/beats/libbeat/common"

type config struct {
	Key    string         `config:"key"`
	Config *common.Config `config:"config"`
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
		Key:    "logs",
		Config: cfg,
	}
}
