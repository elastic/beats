package token

import "github.com/elastic/beats/libbeat/processors"

type config struct {
	TokenPath       string                      `config:"token_path"`
	ConditionConfig *processors.ConditionConfig `config:"condition"`
}

func defaultConfig() config {
	return config{
		TokenPath: "/var/run/secrets/kubernetes.io/serviceaccount/token",
	}
}
