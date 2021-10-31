//+build mage

// This is the build script for Mage. The install target is all you really need.
// The release target is for generating official releases and is really only
// useful to project admins.
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var Aliases = map[string]interface{}{
	"Speak": Say,
}

// Say says something.
func Say(msg string, i int, b bool, d time.Duration) error {
	_, err := fmt.Printf("%v(%T) %v(%T) %v(%T) %v(%T)\n", msg, msg, i, i, b, b, d, d)
	return err
}

// Runs "go install" for mage.  This generates the version info the binary.
func Install() error {
	name := "mage"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}

	gocmd := mg.GoCmd()
	// use GOBIN if set in the environment, otherwise fall back to first path
	// in GOPATH environment string
	bin, err := sh.Output(gocmd, "env", "GOBIN")
	if err != nil {
		return fmt.Errorf("can't determine GOBIN: %v", err)
	}
	if bin == "" {
		gopath, err := sh.Output(gocmd, "env", "GOPATH")
		if err != nil {
			return fmt.Errorf("can't determine GOPATH: %v", err)
		}
		paths := strings.Split(gopath, string([]rune{os.PathListSeparator}))
		bin = filepath.Join(paths[0], "bin")
	}
	// specifically don't mkdirall, if you have an invalid gopath in the first
	// place, that's not on us to fix.
	if err := os.Mkdir(bin, 0700); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create %q: %v", bin, err)
	}
	path := filepath.Join(bin, name)

	// we use go build here because if someone built with go get, then `go
	// install` turns into a no-op, and `go install -a` fails on people's
	// machines that have go installed in a non-writeable directory (such as
	// normal OS installs in /usr/bin)
	return sh.RunV(gocmd, "build", "-o", path, "-ldflags="+flags(), "github.com/magefile/mage")
}

var releaseTag = regexp.MustCompile(`^v1\.[0-9]+\.[0-9]+$`)

// Generates a new release. Expects a version tag in v1.x.x format.
func Release(tag string) (err error) {
	if !releaseTag.MatchString(tag) {
		return errors.New("TAG environment variable must be in semver v1.x.x format, but was " + tag)
	}

	if err := sh.RunV("git", "tag", "-a", tag, "-m", tag); err != nil {
		return err
	}
	if err := sh.RunV("git", "push", "origin", tag); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			sh.RunV("git", "tag", "--delete", "$TAG")
			sh.RunV("git", "push", "--delete", "origin", "$TAG")
		}
	}()
	return sh.RunV("goreleaser")
}

// Remove the temporarily generated files from Release.
func Clean() error {
	return sh.Rm("dist")
}

func flags() string {
	timestamp := time.Now().Format(time.RFC3339)
	hash := hash()
	tag := tag()
	if tag == "" {
		tag = "dev"
	}
	return fmt.Sprintf(`-X "github.com/magefile/mage/mage.timestamp=%s" -X "github.com/magefile/mage/mage.commitHash=%s" -X "github.com/magefile/mage/mage.gitTag=%s"`, timestamp, hash, tag)
}

// tag returns the git tag for the current branch or "" if none.
func tag() string {
	s, _ := sh.Output("git", "describe", "--tags")
	return s
}

// hash returns the git hash for the current repo or "" if none.
func hash() string {
	hash, _ := sh.Output("git", "rev-parse", "--short", "HEAD")
	return hash
}
