package ceph

type Config struct {
	CEPH *CephConfig `config:"ceph"`
}

type CephConfig struct {
        BinaryPath             string `config:"binary_path"	validate:"required"`
        OsdPrefix              string `config:"osd_prefix"	validate:"required"`
        MonPrefix              string `config:"mon_prefix"	validate:"required"`
        SocketDir              string `config:"socket_dir"	validate:"required"`
        SocketSuffix           string `config:"socket_suffix"	validate:"required"`
        User	               string `config:"user"		validate:"required"`
        ConfigPath             string `config:"config_path"	validate:"required"`
}
