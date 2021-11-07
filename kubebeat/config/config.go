// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import "time"

type Config struct {
	KubeConfig string        `config:"kube_config"`
	Period     time.Duration `config:"period"`
}

var DefaultConfig = Config{
	Period: 1 * time.Second,
}
