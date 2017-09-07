package tcp

type TcpConfig struct {
	Host              string `config:"host"`
	Port              int    `config:"port"`
	ReceiveBufferSize int    `config:"receive_buffer_size"`
}

func defaultTcpConfig() TcpConfig {
	return TcpConfig{
		Host:              "localhost",
		Port:              2003,
		ReceiveBufferSize: 1024,
	}
}
