//go:build windows

package npipe

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// administratorSID is the SID for the Administrator user.
const administratorSID = "S-1-5-32-544"

// hasRoot returns true if the user has Administrator/SYSTEM permissions.
func hasRoot() (bool, error) {
	var sid *windows.SID
	// See https://docs.microsoft.com/en-us/windows/desktop/api/securitybaseapi/nf-securitybaseapi-checktokenmembership for more on the api
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if err != nil {
		return false, fmt.Errorf("allocate sid error: %w", err)
	}
	defer func() {
		_ = windows.FreeSid(sid)
	}()

	token := windows.Token(0)

	member, err := token.IsMember(sid)
	if err != nil {
		return false, fmt.Errorf("token membership error: %w", err)
	}

	return member, nil
}
