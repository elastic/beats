package sys

import "golang.org/x/sys/windows"

// PopulateAccount lookups the account name and type associated with a SID.
// The account name, domain, and type are added to the given sid.
func PopulateAccount(sid *SID) error {
	if sid == nil || sid.Identifier == "" {
		return nil
	}

	s, err := windows.StringToSid(sid.Identifier)
	if err != nil {
		return err
	}

	account, domain, accType, err := s.LookupAccount("")
	if err != nil {
		return err
	}

	sid.Name = account
	sid.Domain = domain
	sid.Type = SIDType(accType)
	return nil
}
