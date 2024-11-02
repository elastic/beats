// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package fetcher provides an interface for retrieving identity assets from
// Azure Active Directory.
package fetcher

import (
	"context"

	"github.com/elastic/elastic-agent-libs/logp"
)

// Fetcher provides an interface for retrieving identity assets, such as users
// and groups, from Azure Active Directory.
type Fetcher interface {
	// Groups fetches groups from Azure Active Directory. It may take an
	// optional deltaLink string, which is a URL that can be used to resume
	// from the last query. A slice of Groups and a new delta link may be
	// returned, or an error if a failure occurred.
	Groups(ctx context.Context, deltaLink string) ([]*Group, string, error)

	// Users fetches users from Azure Active Directory. It may take an
	// optional deltaLink string, which is a URL that can be used to resume
	// from the last query. A slice of Users and a new delta link may be
	// returned, or an error if a failure occurred.
	Users(ctx context.Context, deltaLink string) ([]*User, string, error)

	// Devices fetches devices from Azure Active Directory. It may take an
	// optional deltaLink string, which is a URL that can be used to resume
	// from the last query. A slice of Devices and a new delta link may be
	// returned, or an error if a failure occurred.
	Devices(ctx context.Context, deltaLink string) ([]*Device, string, error)

	// SetLogger sets the logger on the Fetcher.
	SetLogger(logger *logp.Logger)
}
