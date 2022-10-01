// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package install

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/gofrs/uuid"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/command"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/distro"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/fileutil"
)

const (
	payloadPath = "Payload"
)

func InstallFromPkg(ctx context.Context, srcPkg, dstDir string, force bool) error {
	dstfp := filepath.Join(dstDir, distro.OsquerydDarwinApp())

	dir, err := installFromCommon(ctx, dstDir, dstfp, force, "pkgutil", "--expand-full", srcPkg)
	// Remove the directory that was created could have been created by pkgutil
	// In case if the process was killed or finished with error but still left a directory behind
	defer os.RemoveAll(dir)

	if err != nil {
		return err
	}

	// Copy over certs
	err = devtools.Copy(filepath.Join(dir, payloadPath, distro.OsquerydCertsDarwinDistroPath()), distro.OsquerydCertsPath(dstDir))
	if err != nil {
		return err
	}

	// Copy over the osqueryd from under Payload into the dstDir directory
	return devtools.Copy(filepath.Join(dir, payloadPath, distro.OsquerydDarwinDistroPath()), filepath.Join(dstDir, distro.OsquerydDarwinApp()))
}

func installFromCommon(ctx context.Context, dstDir, dstfp string, force bool, name string, arg ...string) (dir string, err error) {
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
