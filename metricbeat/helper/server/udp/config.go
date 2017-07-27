package udp

type UdpConfig struct {
	Host              string `config:"host"`
	Port              int    `config:"port"`
	ReceiveBufferSize int    `config:"receive_buffer_size"`
}

func defaultUdpConfig() UdpConfig {
	return UdpConfig{
		Host:              "localhost",
		Port:              2003,
		ReceiveBufferSize: 1024,
	}
}
