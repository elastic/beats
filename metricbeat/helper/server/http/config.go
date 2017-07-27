package http

type HttpConfig struct {
	Host string `config:"host"`
	Port int    `config:"port"`
}

func defaultHttpConfig() HttpConfig {
	return HttpConfig{
		Host: "localhost",
		Port: 8080,
	}
}
