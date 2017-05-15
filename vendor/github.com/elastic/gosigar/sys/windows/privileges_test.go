// +build windows

package windows

import (
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/windows"
)

func TestGetDebugInfo(t *testing.T) {
	debug, err := GetDebugInfo()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v", debug)
}

func TestGetTokenPrivileges(t *testing.T) {
	h, err := windows.GetCurrentProcess()
	if err != nil {
		t.Fatal("GetCurrentProcess", err)
	}

	var token syscall.Token
	err = syscall.OpenProcessToken(syscall.Handle(h), syscall.TOKEN_QUERY, &token)
	if err != nil {
		t.Fatal("OpenProcessToken", err)
	}

	privs, err := GetTokenPrivileges(token)
	if err != nil {
		t.Fatal("GetTokenPrivileges", err)
	}

	for _, priv := range privs {
		t.Log(priv)
	}
}

func TestEnableTokenPrivileges(t *testing.T) {
	h, err := windows.GetCurrentProcess()
	if err != nil {
		t.Fatal("GetCurrentProcess", err)
	}

	var token syscall.Token
	err = syscall.OpenProcessToken(syscall.Handle(h), syscall.TOKEN_ADJUST_PRIVILEGES|syscall.TOKEN_QUERY, &token)
	if err != nil {
		t.Fatal("OpenProcessToken", err)
	}

	privs, err := GetTokenPrivileges(token)
	if err != nil {
		t.Fatal("GetTokenPrivileges", err)
	}

	priv, found := privs[SeDebugPrivilege]
	if found {
		t.Logf("Token has privilege: %v", priv)
	} else {
		t.Logf("Token is missing privilege %v", SeDebugPrivilege)
	}

	err = EnableTokenPrivileges(token, SeDebugPrivilege)
	if err != nil {
		t.Fatal("EnableTokenPrivileges", err)
	}

	privs, err = GetTokenPrivileges(token)
	if err != nil {
		t.Fatal("GetTokenPrivileges", err)
	}

	priv, found = privs[SeDebugPrivilege]
	if found && assert.True(t, priv.Enabled, "%v is not enabled. %v", SeDebugPrivilege, priv) {
		t.Logf("%v is enabled.", SeDebugPrivilege)
	}
}
