package fileset

import (
	"flag"
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	err := flag.Set("strict.perms", "false")
	if err != nil {
		fmt.Fprintln(os.Stderr,
			fmt.Sprintf("failed to set flag strict.perms=false: %v", err))
		os.Exit(1)
	}
	flag.Parse()

	os.Exit(m.Run())
}
