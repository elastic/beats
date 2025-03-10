package testflag

import (
	"flag"
	"fmt"
	"os"
)

// MustSetStrictPermsFalse sets the flag `strict.perms` to false. On error, it
// logs the error to stderr and call os.Exit(1).
func MustSetStrictPermsFalse() {
	err := flag.Set("strict.perms", "false")
	if err != nil {
		fmt.Fprintln(os.Stderr,
			fmt.Sprintf("failed to set flag strict.perms=false: %v", err))
		os.Exit(1)
	}
}
