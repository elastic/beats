package mage

import "github.com/magefile/mage/mg"

const (
	goVersionFilename = "./.go-version"
)

// UpdateGoVersion makes required changes in order to switch to a new version of Go set in `./.go-version`.
func UpdateGoVersion() error {
	mg.Deps(Linter.UpdateGoVersion)
	return nil
}
