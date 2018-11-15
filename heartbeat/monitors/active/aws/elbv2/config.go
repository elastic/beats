package elbv2

type Config struct {
	Name   string   `config:"name"`
	ARNs   []string `config:"arns" validate:"required"`
	Region string   `config:"region" validate:"required"`
}
