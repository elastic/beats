package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func funcWithError() {
	var cmd *cobra.Command
	var args []string
	RunWith(func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("Something bad")
	})(cmd, args)
}

func funcWithoutError() {
	var cmd *cobra.Command
	var args []string
	RunWith(func(cmd *cobra.Command, args []string) error {
		return nil
	})(cmd, args)
}

// Example taken from slides from Andrew Gerrand
// https://talks.golang.org/2014/testing.slide#23
func TestExitWithError(t *testing.T) {
	if os.Getenv("TEST_RUNWITH") == "1" {
		funcWithError()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestExitWithError")
	cmd.Env = append(os.Environ(), "TEST_RUNWITH=1")
	bufError := new(bytes.Buffer)
	cmd.Stderr = bufError
	err := cmd.Run()
	if assert.Error(t, err) {
		assert.Equal(t, err.Error(), "exit status 1")
	}
	assert.Equal(t, "Something bad\n", bufError.String())
}

func TestExitWithoutError(t *testing.T) {
	if os.Getenv("TEST_RUNWITH") == "1" {
		funcWithoutError()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestExitWithoutError")
	cmd.Env = append(os.Environ(), "TEST_RUNWITH=1")
	bufError := new(bytes.Buffer)
	cmd.Stderr = bufError
	err := cmd.Run()
	assert.NoError(t, err)
	assert.Equal(t, "", bufError.String())
}
