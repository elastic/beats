package mage

import (
	"fmt"
	"os"

	"github.com/magefile/mage/sh"
)

func assertUnchanged(path string) error {
	err := sh.Run("git", "diff", "--exit-code", path)
	if err != nil {
		return fmt.Errorf("failed to assert the unchanged file %q: %w", path, err)
	}

	return nil
}

// runWithStdErr runs a command redirecting its stderr to the console instead of discarding it
func runWithStdErr(command string, args ...string) error {
	_, err := sh.Exec(nil, os.Stdout, os.Stderr, command, args...)
	return err
}
