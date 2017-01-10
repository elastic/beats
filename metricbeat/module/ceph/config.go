package ceph

type CephConfig struct {
	BinaryPath   string `config:"binary_path"`
	OsdPrefix    string `config:"osd_prefix"`
	MonPrefix    string `config:"mon_prefix"`
	SocketDir    string `config:"socket_dir"`
	SocketSuffix string `config:"socket_suffix"`
	User         string `config:"user"`
	ConfigPath   string `config:"config_path"`
}

func CheckConfig() CephConfig {

	var userConfig = CephConfig{}

	var defaultConfig = CephConfig{
		BinaryPath:   "/usr/bin/ceph",
		SocketDir:    "/var/run/ceph",
		MonPrefix:    "ceph-mon",
		OsdPrefix:    "ceph-osd",
		SocketSuffix: "asok",
		User:         "client.admin",
		ConfigPath:   "/etc/ceph/ceph.conf",
	}

	if userConfig.BinaryPath == "" {
		userConfig.BinaryPath = defaultConfig.BinaryPath
	}

	if userConfig.SocketDir == "" {
		userConfig.SocketDir = defaultConfig.SocketDir
	}

	if userConfig.MonPrefix == "" {
		userConfig.MonPrefix = defaultConfig.MonPrefix
	}

	if userConfig.OsdPrefix == "" {
		userConfig.OsdPrefix = defaultConfig.OsdPrefix
	}

	if userConfig.SocketSuffix == "" {
		userConfig.SocketSuffix = defaultConfig.SocketSuffix
	}

	if userConfig.User == "" {
		userConfig.User = defaultConfig.User
	}

	if userConfig.ConfigPath == "" {
		userConfig.ConfigPath = defaultConfig.ConfigPath
	}

	return userConfig
}
