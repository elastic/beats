package console

type config struct {
	Pretty bool   `config:"pretty"`
	Format string `config:"format"`
}

var (
	defaultConfig = config{
		Pretty: false,
		Format: "",
	}
)
