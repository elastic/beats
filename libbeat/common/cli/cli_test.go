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

func runCli(testName string) (*bytes.Buffer, error) {
	cmd := exec.Command(os.Args[0], "-test.run="+testName)
	cmd.Env = append(os.Environ(), "TEST_RUNWITH=1")
	stderr := new(bytes.Buffer)
	cmd.Stderr = stderr

	err := cmd.Run()
	return stderr, err
}

// Example taken from slides from Andrew Gerrand
// https://talks.golang.org/2014/testing.slide#23
func TestExitWithError(t *testing.T) {
	if os.Getenv("TEST_RUNWITH") == "1" {
		func() {
			var cmd *cobra.Command
			var args []string
			RunWith(func(cmd *cobra.Command, args []string) error {
				return fmt.Errorf("Something bad")
			})(cmd, args)
			return
		}()
	}

	stderr, err := runCli("TestExitWithError")
	if assert.Error(t, err) {
		assert.Equal(t, err.Error(), "exit status 1")
	}
	assert.Equal(t, "Something bad\n", stderr.String())
}

func TestExitWithoutError(t *testing.T) {
	if os.Getenv("TEST_RUNWITH") == "1" {
		func() {
			var cmd *cobra.Command
			var args []string
			RunWith(func(cmd *cobra.Command, args []string) error {
				return nil
			})(cmd, args)
		}()
		return
	}

	stderr, err := runCli("TestExitWithoutError")
	assert.NoError(t, err)
	assert.Equal(t, "", stderr.String())
}

func TestExitWithPanic(t *testing.T) {
	if os.Getenv("TEST_RUNWITH") == "1" {
		func() {
			var cmd *cobra.Command
			var args []string
			RunWith(func(cmd *cobra.Command, args []string) error {
				panic("something really bad happened")
			})(cmd, args)
		}()
		return
	}

	stderr, err := runCli("TestExitWithPanic")
	if assert.Error(t, err) {
		assert.Equal(t, err.Error(), "exit status 1")
	}
	assert.Contains(t, stderr.String(), "something really bad happened")
}
