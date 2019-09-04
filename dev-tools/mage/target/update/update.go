package update

import "github.com/magefile/mage/sh"

// Update updates the generated files (aka make update).
func Update() error {
	return sh.Run("make", "update")
}
