//+build ignore

package main

import (
	"os"

	"github.com/magefile/mage/mage"
)

// This is a bootstrap builder, to build mage when you don't already *have* mage.
// Run it like
// go run bootstrap.go
// and it will install mage with all the right flags created for you.

func main() {
	os.Args = []string{os.Args[0], "-v", "install"}
	os.Exit(mage.Main())
}
