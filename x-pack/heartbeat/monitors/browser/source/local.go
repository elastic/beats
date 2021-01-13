package source

import (
	"fmt"
	"io/ioutil"
	"github.com/otiai10/copy"
)

type LocalSource struct {
	OrigPath     string                `config:"path"`
	workingPath string
	BaseSource
}

func (l *LocalSource) Fetch() (err error) {
	l.workingPath, err = ioutil.TempDir("/tmp", "elastic-synthetics-")
	if err != nil {
		return fmt.Errorf("could not create tmp dir: %w", err)
	}

	err = copy.Copy(l.OrigPath, l.workingPath)
	if err != nil {
		return fmt.Errorf("could not copy suite: %w", err)
	}
	return nil
}

func (l *LocalSource) Workdir() string {
	return l.workingPath
}
