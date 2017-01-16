package ceph

type CephConfig struct {
	BinaryPath string `config:"binary_path"`
	User       string `config:"user"`
	ConfigPath string `config:"config_path"`
}

func CheckConfig() CephConfig {

	var userConfig = CephConfig{}

	var defaultConfig = CephConfig{
		BinaryPath: "/usr/bin/ceph",
		User:       "client.admin",
		ConfigPath: "/etc/ceph/ceph.conf",
	}

	if userConfig.BinaryPath == "" {
		userConfig.BinaryPath = defaultConfig.BinaryPath
	}

	if userConfig.User == "" {
		userConfig.User = defaultConfig.User
	}

	if userConfig.ConfigPath == "" {
		userConfig.ConfigPath = defaultConfig.ConfigPath
	}

	return userConfig
}
