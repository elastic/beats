// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hostusers

type hostUser struct {
	UID         int64  `osquery:"uid" desc:"User ID"`
	GID         int64  `osquery:"gid" desc:"Group ID (unsigned)"`
	UIDSigned   int64  `osquery:"uid_signed" desc:"User ID as int64 signed (Apple)"`
	GIDSigned   int64  `osquery:"gid_signed" desc:"Default group ID as int64 signed (Apple)"`
	Username    string `osquery:"username" desc:"Username"`
	Description string `osquery:"description" desc:"Optional user description"`
	Directory   string `osquery:"directory" desc:"User's home directory"`
	Shell       string `osquery:"shell" desc:"User's configured default shell"`
	UUID        string `osquery:"uuid" desc:"User's UUID (Apple) or SID (Windows)"`
}
