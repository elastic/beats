// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package mage

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"
)

type dmgBuilder struct {
	PackageSpec

	SigningInfo           *AppleSigningInfo
	Identifier            string
	PreferencePaneDir     string
	PreferencePanePkgFile string
	InternalBeatPkg       string
	BeatPkg               string

	beatsDir string
	dmgDir   string

	// Build tools.
	pkgbuild     func(args ...string) error
	productbuild func(args ...string) error
	spctl        func(args ...string) error
	codesign     func(args ...string) error
	hdiutil      func(args ...string) error
}

func newDMGBuilder(spec PackageSpec) (*dmgBuilder, error) {
	for _, cmd := range []string{"pkgbuild", "productbuild", "spctl", "codesign", "hdiutil"} {
		if _, err := exec.LookPath(cmd); err != nil {
			return nil, errors.Wrapf(err, "required tool '%v' for DMG packaging not found on PATH", cmd)
		}
	}

	beatsDir, err := ElasticBeatsDir()
	if err != nil {
		return nil, err
	}

	preferencePaneDir := filepath.Join(beatsDir, "dev-tools/packaging/preference-pane")
	preferencePanePkgFile := filepath.Join(preferencePaneDir, "build/BeatsPrefPane.pkg")
	beatIdentifier, ok := spec.evalContext["identifier"].(string)
	if !ok {
		return nil, errors.Errorf("identifier not specified for DMG packaging")
	}

	spec.OutputFile, err = spec.Expand(defaultBinaryName)
	if err != nil {
		return nil, err
	}

	info, err := GetAppleSigningInfo()
	if err != nil {
		return nil, err
	}

	return &dmgBuilder{
		PackageSpec:           spec,
		SigningInfo:           info,
		Identifier:            beatIdentifier,
		PreferencePaneDir:     preferencePaneDir,
		PreferencePanePkgFile: preferencePanePkgFile,

		beatsDir: beatsDir,
		dmgDir:   filepath.Join(spec.packageDir, "dmg"),

		pkgbuild:     sh.RunCmd("pkgbuild"),
		productbuild: sh.RunCmd("productbuild"),
		spctl:        sh.RunCmd("spctl", "-a", "-t"),
		codesign:     sh.RunCmd("codesign"),
		hdiutil:      sh.RunCmd("hdiutil"),
	}, nil
}

// Create .pkg for preference pane.
func (b *dmgBuilder) buildPreferencePane() error {
	return errors.Wrap(Mage(b.PreferencePaneDir), "failed to build Beats preference pane")
}

func (b *dmgBuilder) buildBeatPkg() error {
	beatPkgRoot := filepath.Join(b.packageDir, "beat-pkg-root")
	if err := os.RemoveAll(beatPkgRoot); err != nil {
		return errors.Wrap(err, "failed to clean beat-pkg-root")
	}

	// Copy files into the packaging root and set their mode.
	for _, f := range b.Files {
		target := filepath.Join(beatPkgRoot, f.Target)
		if err := Copy(f.Source, target); err != nil {
			return err
		}

		info, err := os.Stat(target)
		if err != nil {
			return err
		}

		if info.Mode().IsRegular() && info.Mode().Perm() != f.Mode {
			if err = os.Chmod(target, f.Mode); err != nil {
				return err
			}
		}
	}

	b.InternalBeatPkg = filepath.Join(b.packageDir, "pkgs", "internal-"+b.OutputFile+".pkg")

	args := []string{
		"--root", beatPkgRoot,
		"--scripts", filepath.Join(b.packageDir, "scripts"),
		"--identifier", b.Identifier,
		"--version", b.MustExpand("{{.Version}}{{if .Snapshot}}-SNAPSHOT{{end}}"),
	}
	if b.SigningInfo.Sign {
		args = append(args, "--sign", b.SigningInfo.Installer.ID, "--timestamp")
	}
	args = append(args, createDir(b.InternalBeatPkg))
	if err := b.pkgbuild(args...); err != nil {
		return err
	}

	return nil
}

func (b *dmgBuilder) buildProductPkg() error {
	var (
		distributionPlist = filepath.Join(b.packageDir, "distributions.plist")
		resourcesDir      = filepath.Join(b.packageDir, "resources")
	)

	b.MustExpandFile(
		filepath.Join(b.beatsDir, "dev-tools/packaging/templates/darwin/distribution.plist.tmpl"),
		distributionPlist)
	b.MustExpandFile(
		filepath.Join(b.beatsDir, "dev-tools/packaging/templates/darwin/README.html.tmpl"),
		filepath.Join(resourcesDir, "README.html"))
	for t, pf := range b.Files {
		if strings.HasSuffix(t, "LICENSE.txt") {
			Copy(pf.Source, filepath.Join(resourcesDir, "LICENSE.txt"))
			break
		}
	}
	b.MustExpandFile(
		filepath.Join(b.beatsDir, "dev-tools/packaging/templates/darwin/README.html.tmpl"),
		filepath.Join(resourcesDir, "README.html"))

	if err := os.RemoveAll(b.dmgDir); err != nil {
		return err
	}
	b.BeatPkg = filepath.Join(b.dmgDir, b.OutputFile+".pkg")

	// Create .pkg containing the previous two .pkg files.
	args := []string{
		"--distribution", distributionPlist,
		"--resources", resourcesDir,
		"--package-path", filepath.Dir(b.InternalBeatPkg),
		"--package-path", filepath.Dir(b.PreferencePanePkgFile),
		"--component-compression", "auto",
	}
	if b.SigningInfo.Sign {
		args = append(args, "--sign", b.SigningInfo.Installer.ID, "--timestamp")
	}
	args = append(args, createDir(b.BeatPkg))
	if err := b.productbuild(args...); err != nil {
		return err
	}

	if b.SigningInfo.Sign {
		if err := b.spctl("install", b.BeatPkg); err != nil {
			return err
		}
	}

	return nil
}

func (b *dmgBuilder) buildUninstallApp() error {
	const (
		uninstallerIcons = "Uninstall.app/Contents/Resources/uninstaller.icns"
		uninstallScript  = "Uninstall.app/Contents/MacOS/uninstall.sh"
		infoPlist        = "Uninstall.app/Contents/Info.plist"
	)

	inputDir := filepath.Join(b.beatsDir, "dev-tools/packaging/templates/darwin/dmg")

	Copy(
		filepath.Join(inputDir, uninstallerIcons),
		filepath.Join(b.dmgDir, uninstallerIcons),
	)
	b.MustExpandFile(
		filepath.Join(inputDir, infoPlist+".tmpl"),
		filepath.Join(b.dmgDir, infoPlist),
	)
	b.MustExpandFile(
		filepath.Join(inputDir, uninstallScript+".tmpl"),
		filepath.Join(b.dmgDir, uninstallScript),
	)
	if err := os.Chmod(filepath.Join(b.dmgDir, uninstallScript), 0755); err != nil {
		return err
	}

	if b.SigningInfo.Sign {
		uninstallApp := filepath.Join(b.dmgDir, "Uninstall.app")
		if err := b.codesign("-s", b.SigningInfo.App.ID, "--timestamp", uninstallApp); err != nil {
			return err
		}

		if err := b.spctl("exec", uninstallApp); err != nil {
			return err
		}
	}

	return nil
}

// Create a .dmg file containing both the Uninstall.app and .pkg file.
func (b *dmgBuilder) buildDMG() error {
	dmgFile := filepath.Join(distributionsDir, DMG.AddFileExtension(b.OutputFile))

	args := []string{
		"create",
		"-volname", b.MustExpand("{{.BeatName | title}} {{.Version}}{{if .Snapshot}}-SNAPSHOT{{end}}"),
		"-srcfolder", b.dmgDir,
		"-ov",
		createDir(dmgFile),
	}
	if err := b.hdiutil(args...); err != nil {
		return err
	}

	// Sign the .dmg.
	if b.SigningInfo.Sign {
		if err := b.codesign("-s", b.SigningInfo.App.ID, "--timestamp", dmgFile); err != nil {
			return err
		}

		if err := b.spctl("install", dmgFile); err != nil {
			return err
		}
	}

	return errors.Wrap(CreateSHA512File(dmgFile), "failed to create .sha512 file")
}

func (b *dmgBuilder) Build() error {
	// Mark this function as a dep so that is is only invoked once.
	mg.Deps(b.buildPreferencePane)

	var err error
	if err = b.buildBeatPkg(); err != nil {
		return errors.Wrap(err, "failed to build internal beat pkg")
	}
	if err = b.buildProductPkg(); err != nil {
		return errors.Wrap(err, "failed to build beat product pkg (pref pane + beat)")
	}
	if err = b.buildUninstallApp(); err != nil {
		return errors.Wrap(err, "failed to build Uninstall.app")
	}
	if err = b.buildDMG(); err != nil {
		return errors.Wrap(err, "failed to build beat dmg")
	}
	return nil
}
