package beater

import (
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"

	"{{cookiecutter.beat_path}}/{{cookiecutter.beat}}/config"
)

type {{cookiecutter.beat|capitalize}} struct {
	done chan struct{}
}

// Creates beater
func New() *{{cookiecutter.beat|capitalize}} {
	return &{{cookiecutter.beat|capitalize}}{
		done: make(chan struct{}),
	}
}

/// *** Beater interface methods ***///

func (bt *{{cookiecutter.beat|capitalize}}) Config(b *beat.Beat) error {

	cfg := &config.{{cookiecutter.beat|capitalize}}Config{}
	err := cfgfile.Read(&cfg, "")
	if err != nil {
		return fmt.Errorf("Error reading config file: %v", err)
	}

	<-bt.done

	return nil
}

func (bt *{{cookiecutter.beat|capitalize}}) Setup(b *beat.Beat) error {
	return nil
}

func (bt *{{cookiecutter.beat|capitalize}}) Run(b *beat.Beat) error {
	fmt.Println("{{cookiecutter.beat}} is running! Hit CTRL-C to stop it.")
	return nil
}

func (bt *{{cookiecutter.beat|capitalize}}) Cleanup(b *beat.Beat) error {
	return nil
}

func (bt *{{cookiecutter.beat|capitalize}}) Stop() {
	close(bt.done)
}
