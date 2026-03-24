// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package bundled

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/distro"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/install/artifactformat"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/msiutil"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/pkgutil"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/tar"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/zip"
)

type Plan struct {
	OSQuerySourcePath string
	CertsSourcePath   string
	LensesSourcePath  string
	OSQueryTargetPath string
}

func BuildPlan(osarch distro.OSArch, format artifactformat.Format, installDir string) (Plan, error) {
	switch format {
	case artifactformat.TarGz:
		return Plan{
			OSQuerySourcePath: distro.OsquerydLinuxDistroPath(),
			CertsSourcePath:   distro.OsquerydCertsLinuxDistroPath(),
			LensesSourcePath:  distro.OsquerydLensesLinuxDistroDir(),
			OSQueryTargetPath: distro.OsquerydPath(installDir),
		}, nil
	case artifactformat.Pkg:
		return Plan{
			OSQuerySourcePath: distro.OsquerydDarwinDistroPath(),
			CertsSourcePath:   distro.OsquerydCertsDarwinDistroPath(),
			LensesSourcePath:  distro.OsquerydLensesDarwinDistroDir(),
			OSQueryTargetPath: filepath.Join(installDir, distro.OsquerydDarwinApp()),
		}, nil
	case artifactformat.Msi:
		return Plan{
			OSQuerySourcePath: filepath.Join("osquery", "osqueryd", "osqueryd.exe"),
			CertsSourcePath:   distro.OsquerydCertsWindowsDistroPath(),
			OSQueryTargetPath: distro.OsquerydPathForOS(osarch.OS, installDir),
		}, nil
	case artifactformat.Zip:
		return Plan{
			OSQuerySourcePath: distro.OsquerydWindowsZipPath(),
			CertsSourcePath:   distro.OsquerydCertsWindowsZipDistroPath(),
			OSQueryTargetPath: distro.OsquerydPathForOS(osarch.OS, installDir),
		}, nil
	default:
		return Plan{}, fmt.Errorf("unsupported artifact format %q", format)
	}
}

func BuildRuntimePlan(extractedDir, sourceBinDir, goos, targetDir string) (Plan, error) {
	sourceBinPath := osqd.OsquerydPathForPlatform(goos, sourceBinDir)
	var targetBinPath string
	if goos == "darwin" {
		sourceBinPath = filepath.Dir(filepath.Dir(filepath.Dir(sourceBinPath)))
		targetBinPath = filepath.Dir(filepath.Dir(filepath.Dir(osqd.OsquerydPathForPlatform(goos, targetDir))))
	} else {
		targetBinPath = osqd.OsquerydPathForPlatform(goos, targetDir)
	}
	relSourceBin, err := filepath.Rel(extractedDir, sourceBinPath)
	if err != nil {
		return Plan{}, err
	}

	var certsRel string
	if certsSource, ok := FindFirstPathByBase(extractedDir, "certs.pem", false); ok {
		if certsRel, err = filepath.Rel(extractedDir, certsSource); err != nil {
			return Plan{}, err
		}
	}
	var lensesRel string
	if lensesSource, ok := FindLensesDir(extractedDir); ok {
		if lensesRel, err = filepath.Rel(extractedDir, lensesSource); err != nil {
			return Plan{}, err
		}
	}

	return Plan{
		OSQuerySourcePath: relSourceBin,
		OSQueryTargetPath: targetBinPath,
		CertsSourcePath:   certsRel,
		LensesSourcePath:  lensesRel,
	}, nil
}

func ExtractToTemp(format artifactformat.Format, src, tmpdir string, plan *Plan) error {
	if plan == nil {
		return artifactformat.ExtractAll(format, src, tmpdir)
	}
	switch format {
	case artifactformat.TarGz:
		paths := []string{plan.OSQuerySourcePath}
		if plan.CertsSourcePath != "" {
			paths = append(paths, plan.CertsSourcePath)
		}
		if plan.LensesSourcePath != "" {
			paths = append(paths, plan.LensesSourcePath)
		}
		return tar.ExtractFile(src, tmpdir, paths...)
	case artifactformat.Pkg:
		return pkgutil.Expand(src, tmpdir)
	case artifactformat.Msi:
		return msiutil.Expand(src, tmpdir)
	case artifactformat.Zip:
		if plan.CertsSourcePath != "" {
			return zip.UnzipFile(src, tmpdir, plan.OSQuerySourcePath, plan.CertsSourcePath)
		}
		return zip.UnzipFile(src, tmpdir, plan.OSQuerySourcePath)
	default:
		return fmt.Errorf("unsupported artifact format %q", format)
	}
}

func InstallFromTemp(tmpdir, installDir string, plan Plan, copyFn func(src, dst string) error) error {
	if plan.CertsSourcePath != "" {
		certsDir := filepath.Dir(distro.OsquerydCertsPath(installDir))
		if err := os.MkdirAll(certsDir, 0750); err != nil {
			return err
		}
		if err := copyFn(filepath.Join(tmpdir, plan.CertsSourcePath), distro.OsquerydCertsPath(installDir)); err != nil {
			return err
		}
	}

	if plan.LensesSourcePath != "" {
		lensesDir := distro.OsquerydLensesDir(installDir)
		if err := os.MkdirAll(lensesDir, 0750); err != nil {
			return err
		}
		if err := copyFn(filepath.Join(tmpdir, plan.LensesSourcePath), lensesDir); err != nil {
			return err
		}
	}

	return copyFn(filepath.Join(tmpdir, plan.OSQuerySourcePath), plan.OSQueryTargetPath)
}

func FindFirstPathByBase(root, base string, dirOnly bool) (string, bool) {
	var candidates []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // best-effort search, skip inaccessible entries
		}
		if filepath.Base(path) != base {
			return nil
		}
		if dirOnly && !d.IsDir() {
			return nil
		}
		if !dirOnly && d.IsDir() {
			return nil
		}
		candidates = append(candidates, path)
		return nil
	})
	if len(candidates) == 0 {
		return "", false
	}
	sort.Slice(candidates, func(i, j int) bool {
		return len(candidates[i]) < len(candidates[j])
	})
	return candidates[0], true
}

func FindLensesDir(root string) (string, bool) {
	var candidates []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil //nolint:nilerr // best-effort search, skip inaccessible entries
		}
		if filepath.Base(path) != "lenses" {
			return nil
		}
		candidates = append(candidates, path)
		return nil
	})
	if len(candidates) == 0 {
		return "", false
	}
	sort.Slice(candidates, func(i, j int) bool {
		pi := strings.Contains(candidates[i], filepath.Join("share", "osquery", "lenses"))
		pj := strings.Contains(candidates[j], filepath.Join("share", "osquery", "lenses"))
		if pi != pj {
			return pi
		}
		return len(candidates[i]) < len(candidates[j])
	})
	return candidates[0], true
}

func CopyPath(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFileOrSymlink(src, dst, info)
}

func copyDir(srcDir, dstDir string) error {
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dstDir, relPath)
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(targetPath, info.Mode().Perm())
		}
		return copyFileOrSymlink(path, targetPath, info)
	})
}

func copyFileOrSymlink(src, dst string, info os.FileInfo) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0750); err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		_ = os.Remove(dst)
		return os.Symlink(target, dst)
	}
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	return err
}
