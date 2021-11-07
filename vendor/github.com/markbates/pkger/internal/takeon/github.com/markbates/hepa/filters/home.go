package filters

import (
	"bytes"
	"os"
)

var home = func() dir {
	var d dir
	home, ok := os.LookupEnv("HOME")
	if !ok {
		pwd, err := os.Getwd()
		if err != nil {
			d.Err = err
			return d
		}
		home = pwd
	}
	d.Dir = home

	return d
}()

func Home() FilterFunc {
	return func(b []byte) ([]byte, error) {
		if home.Err != nil {
			return b, home.Err
		}
		return bytes.ReplaceAll(b, []byte(home.Dir), []byte("$HOME")), nil
	}
}
