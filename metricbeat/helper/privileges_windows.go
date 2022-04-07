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

package helper

import (
	"sync"
	"syscall"

	"github.com/pkg/errors"

	"github.com/elastic/gosigar/sys/windows"

	"github.com/elastic/beats/v8/libbeat/logp"
)

var once sync.Once

// errMissingSeDebugPrivilege indicates that the SeDebugPrivilege is not
// present in the process's token. This is distinct from disabled. The token
// would be missing if the user does not have "Debug programs" rights. By
// default, only administrators and LocalSystem accounts have the privileges to
// debug programs.
var errMissingSeDebugPrivilege = errors.New("Metricbeat is running without " +
	"SeDebugPrivilege, a Windows privilege that allows it to collect metrics " +
	"from other processes. The user running Metricbeat may not have the " +
	"appropriate privileges or the security policy disallows it.")

// enableSeDebugPrivilege enables the SeDebugPrivilege if it is present in
// the process's token.
func enableSeDebugPrivilege() error {
	self, err := syscall.GetCurrentProcess()
	if err != nil {
		return err
	}

	var token syscall.Token
	err = syscall.OpenProcessToken(self, syscall.TOKEN_QUERY|syscall.TOKEN_ADJUST_PRIVILEGES, &token)
	if err != nil {
		return err
	}

	if err = windows.EnableTokenPrivileges(token, windows.SeDebugPrivilege); err != nil {
		return errors.Wrap(err, "EnableTokenPrivileges failed")
	}

	return nil
}

// CheckAndEnableSeDebugPrivilege checks if the process's token has the
// SeDebugPrivilege and enables it if it is disabled.
func CheckAndEnableSeDebugPrivilege() error {
	var err error
	once.Do(func() {
		err = checkAndEnableSeDebugPrivilege()
	})
	return err
}

func checkAndEnableSeDebugPrivilege() error {
	info, err := windows.GetDebugInfo()
	if err != nil {
		return errors.Wrap(err, "GetDebugInfo failed")
	}
	logp.Info("Metricbeat process and system info: %v", info)

	seDebug, found := info.ProcessPrivs[windows.SeDebugPrivilege]
	if !found {
		return errMissingSeDebugPrivilege
	}

	if seDebug.Enabled {
		logp.Info("SeDebugPrivilege is enabled. %v", seDebug)
		return nil
	}

	if err = enableSeDebugPrivilege(); err != nil {
		logp.Warn("Failure while attempting to enable SeDebugPrivilege. %v", err)
	}

	info, err = windows.GetDebugInfo()
	if err != nil {
		return errors.Wrap(err, "GetDebugInfo failed")
	}

	seDebug, found = info.ProcessPrivs[windows.SeDebugPrivilege]
	if !found {
		return errMissingSeDebugPrivilege
	}

	if !seDebug.Enabled {
		return errors.Errorf("Metricbeat failed to enable the "+
			"SeDebugPrivilege, a Windows privilege that allows it to collect "+
			"metrics from other processes. %v", seDebug)
	}

	logp.Info("SeDebugPrivilege is now enabled. %v", seDebug)
	return nil
}
