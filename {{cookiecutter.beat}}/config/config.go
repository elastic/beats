// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

type {{cookiecutter.beat|capitalize}}Config struct {
	Period string `yaml:"period"`
}
