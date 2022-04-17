// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package install

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"

	"github.com/gofrs/uuid"

	devtools "github.com/menderesk/beats/v7/dev-tools/mage"

	"github.com/menderesk/beats/v7/x-pack/osquerybeat/internal/command"
	"github.com/menderesk/beats/v7/x-pack/osquerybeat/internal/distro"
	"github.com/menderesk/beats/v7/x-pack/osquerybeat/internal/fileutil"
)

func InstallFromPkg(ctx context.Context, srcPkg, dstDir string, force bool) error {
	dstfp := filepath.Join(dstDir, distro.OsquerydDarwinApp())

	dir, err := installFromCommon(ctx, srcPkg, dstDir, dstfp, force, "pkgutil", "--expand-full", srcPkg)
	// Remove the directory that was created could have been created by pkgutil
	// In case if the process was killed or finished with error but still left a directory behind
	defer os.RemoveAll(dir)

	if err != nil {
		return err
	}

	// Copy over the osqueryd from under Payload into the dstDir directory
	return devtools.Copy(filepath.Join(dir, "Payload", distro.OsquerydDarwinDistroPath()), filepath.Join(dstDir, distro.OsquerydDarwinApp()))
}

func InstallFromMSI(ctx context.Context, srcMSI, dstDir string, force bool) error {
	dstfp := filepath.Join(dstDir, distro.OsquerydFilename())

	// Winderz is odd, passing params to msiexec as usual didn't work
	dir, err := installFromCommon(ctx, srcMSI, dstDir, dstfp, force, "msiexec", `/quiet /a "`+srcMSI+`"`)

	// Remove the directory that was created could have been created by msiexec
	// In case if the process was killed or finished with error but still left a directory behind
	defer os.RemoveAll(dir)

	if err != nil {
		return err
	}

	// Copy over the or osquery.app osqueryd from under osquery/osqueryd into the dstDir directory
	return devtools.Copy(path.Join(dir, "osquery", distro.OsquerydPath("osqueryd")), dstfp)
}

func installFromCommon(ctx context.Context, srcfp, dstDir, dstfp string, force bool, name string, arg ...string) (dir string, err error) {
	if !force {
		//check if files exists
		exists, err := fileutil.FileExists(dstfp)
		if err != nil {
			return dir, err
		}
		if exists {
			return dir, nil
		}
	}

	if err := os.MkdirAll(dstDir, 0750); err != nil {
		return dir, fmt.Errorf("failed to create dir %v, %w", dstDir, err)
	}

	// Temp directory for extracting the .pkg or .msi
	uid := uuid.Must(uuid.NewV4()).String()
	dir = filepath.Join(dstDir, uid)

	if runtime.GOOS == "darwin" {
		arg = append(arg, dir)
		// Extract .pkg
		_, err = command.Execute(ctx, name, arg...)
		return dir, err
	}

	// Extract .msi
	idx := len(arg) - 1
	arg[idx] = arg[idx] + ` TARGETDIR="` + dir + `"`
	cmd := exec.Command(name)

	// Set directly to avoid args escaping
	setCommandArg(cmd, arg[idx])

	return dir, cmd.Run()
}
