package main

import (
	"fmt"
	"os"

	"github.com/mattn/go-zglob"
)

func main() {
	for _, arg := range os.Args[1:] {
		matches, err := zglob.Glob(arg)
		if err != nil {
			continue
		}
		for _, m := range matches {
			if fi, err := os.Stat(m); err == nil && fi.Mode().IsRegular() {
				fmt.Println(m)
			}
		}
	}
}
