package system

import (
	"syscall"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/gosigar/sys/windows"
	"github.com/pkg/errors"
)

// errMissingSeDebugPrivilege indicates that the SeDebugPrivilege is not
// present in the process's token. This is distinct from disabled. The token
// would be missing if the user does not have "Debug programs" rights. By
// default, only administrators and LocalSystem accounts have the privileges to
// debug programs.
var errMissingSeDebugPrivilege = errors.New("Metricbeat is running without " +
	"SeDebugPrivilege, a Windows privilege that allows it to collect metrics " +
	"from other processes. The user running Metricbeat may not have the " +
	"appropriate privileges or the security policy disallows it.")

func initModule() {
	if err := checkAndEnableSeDebugPrivilege(); err != nil {
		logp.Warn("%v", err)
	}
}

// checkAndEnableSeDebugPrivilege checks if the process's token has the
// SeDebugPrivilege and enables it if it is disabled.
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
