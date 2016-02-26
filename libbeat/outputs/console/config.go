package console

type config struct {
	Pretty bool `config:"pretty"`
}

var (
	defaultConfig = config{
		Pretty: false,
	}
)
