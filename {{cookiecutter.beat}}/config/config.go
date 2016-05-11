// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

type Config struct {
	{{cookiecutter.beat|capitalize}} {{cookiecutter.beat|capitalize}}Config
}

type {{cookiecutter.beat|capitalize}}Config struct {
	Period string `config:"period"`
}
