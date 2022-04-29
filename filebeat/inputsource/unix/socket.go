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

package unix

import (
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strconv"

	"github.com/pkg/errors"

	"github.com/elastic/elastic-agent-libs/logp"
)

func cleanupStaleSocket(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		// If the file does not exist, then the cleanup can be considered successful.
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Wrapf(err, "cannot lstat unix socket file at location %s", path)
	}

	if runtime.GOOS != "windows" {
		// see https://github.com/golang/go/issues/33357 for context on Windows socket file attributes bug
		if info.Mode()&os.ModeSocket == 0 {
			return fmt.Errorf("refusing to remove file at location %s, it is not a socket", path)
		}
	}

	if err := os.Remove(path); err != nil {
		return errors.Wrapf(err, "cannot remove existing unix socket file at location %s", path)
	}

	return nil
}

func setSocketOwnership(path string, group *string) error {
	if group != nil {
		if runtime.GOOS == "windows" {
			logp.NewLogger("unix").Warn("windows does not support the 'group' configuration option, ignoring")
			return nil
		}
		g, err := user.LookupGroup(*group)
		if err != nil {
			return err
		}
		gid, err := strconv.Atoi(g.Gid)
		if err != nil {
			return err
		}
		return os.Chown(path, -1, gid)
	}
	return nil
}

func setSocketMode(path string, mode *string) error {
	if mode != nil {
		m, err := parseFileMode(*mode)
		if err != nil {
			return err
		}
		return os.Chmod(path, m)
	}
	return nil
}

func parseFileMode(mode string) (os.FileMode, error) {
	parsed, err := strconv.ParseUint(mode, 8, 32)
	if err != nil {
		return 0, err
	}
	if parsed > 0o777 {
		return 0, errors.New("invalid file mode")
	}
	return os.FileMode(parsed), nil
}
