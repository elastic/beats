package util

import (
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type ConfigYaml struct {
	Query  []InnerConfig     `yaml:"hardware_query"`
	Format InnerConfigFormat `yaml:"output_format"`
}

type InnerConfig struct {
	TypeOf string `yaml:"type"`
	Name   string `yaml:"name"`
}

type InnerConfigFormat struct {
	UseType  bool `yaml:"use_type_as_key"`
	UseConst bool `yaml:"use_constant_key"`
}

func ReadFile(cfg *ConfigYaml) {
	f, err := os.Open("hardware.yml")
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(cfg)
	if err != nil {
		log.Println(err)
	}
}

func B2s(bs []int8) string {
	b := make([]byte, len(bs))
	for i, v := range bs {
		b[i] = byte(v)
	}
	return string(b)
}
