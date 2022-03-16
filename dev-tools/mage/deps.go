package mage

import (
	"fmt"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Deps contains targets related to checking dependencies
type Deps mg.Namespace

// CheckModuleTidy checks if `go mod tidy` was run before the last commit.
func (Deps) CheckModuleTidy() error {
	err := sh.Run("go", "mod", "tidy")
	if err != nil {
		return err
	}
	err = assertUnchanged("go.mod")
	if err != nil {
		return fmt.Errorf("`go mod tidy` was not called before the last commit: %w", err)
	}

	return nil
}
