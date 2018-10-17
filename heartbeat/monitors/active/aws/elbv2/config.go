package elbv2

type Config struct {
	Name string `config:"name"`

	ARNs   []string `config:"urls" validate:"required"`
	Region string   `config:"region" validate:"required"`
}
