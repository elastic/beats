package membroker

type config struct {
	Events int `config:"events" validate:"min=32"`
}

var defaultConfig = config{
	Events: 4096,
}
