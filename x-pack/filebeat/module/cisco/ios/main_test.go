package ios

import (
	"flag"
	"os"
	"testing"

	"github.com/elastic/beats/v7/testing/testflag"
)

func TestMain(m *testing.M) {
	testflag.MustSetStrictPermsFalse()

	flag.Parse()

	os.Exit(m.Run())
}
