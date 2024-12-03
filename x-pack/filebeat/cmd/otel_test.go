package cmd

import (
	"fmt"
	"testing"
)

func TestOtel(t *testing.T) {
	cmd := OtelCmd()
	fmt.Println(cmd.RunE(cmd, []string{}))
}
