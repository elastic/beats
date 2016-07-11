// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import "time"

type Config struct {
	{{cookiecutter.beat|capitalize}} {{cookiecutter.beat|capitalize}}Config
}

type {{cookiecutter.beat|capitalize}}Config struct {
	Period time.Duration `config:"period"`
}

var DefaultConfig = Config{
	{{cookiecutter.beat|capitalize}}: {{cookiecutter.beat|capitalize}}Config{
		Period: 1 * time.Second,
	},
}
