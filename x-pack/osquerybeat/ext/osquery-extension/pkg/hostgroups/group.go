// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hostgroups

type hostGroup struct {
	GID       int64  `osquery:"gid" desc:"Unsigned int64 group ID"`
	GIDSigned int64  `osquery:"gid_signed" desc:"A signed int64 version of gid"`
	GroupName string `osquery:"groupname" desc:"Canonical local group name"`
}
