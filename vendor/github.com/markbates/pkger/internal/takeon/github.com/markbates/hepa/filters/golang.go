package filters

import (
	"bytes"
	"os"
	"path/filepath"
)

func Golang() FilterFunc {
	return func(b []byte) ([]byte, error) {
		gp, err := gopath(home)
		if err != nil {
			return nil, err
		}

		b = bytes.ReplaceAll(b, []byte(gp.Dir), []byte("$GOPATH"))

		gru, err := goroot(gp)
		if err != nil {
			return nil, err
		}
		b = bytes.ReplaceAll(b, []byte(gru.Dir), []byte("$GOROOT"))
		return b, nil
	}
}

func goroot(gp dir) (dir, error) {
	gru, ok := os.LookupEnv("GOROOT")
	if !ok {
		if gp.Err != nil {
			return gp, gp.Err
		}
		gru = filepath.Join(string(gp.Dir), "go")
	}
	return dir{
		Dir: gru,
	}, nil
}

func gopath(home dir) (dir, error) {
	gp, ok := os.LookupEnv("GOPATH")
	if !ok {
		if home.Err != nil {
			return home, home.Err
		}
		gp = filepath.Join(string(home.Dir), "go")
	}
	return dir{
		Dir: gp,
	}, nil
}
