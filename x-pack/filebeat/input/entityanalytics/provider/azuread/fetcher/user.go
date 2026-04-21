// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fetcher

import (
	"github.com/gofrs/uuid/v5"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/collections"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// User represents a user identity asset.
type User struct {
	// The ID (UUIDv4) of the user.
	ID uuid.UUID `json:"id"`
	// The attributes for the user.
	Fields mapstr.M `json:"fields"`
	// A set of UUIDs which are groups this user is a member of.
	MemberOf collections.UUIDSet `json:"memberOf"`
	// A set of UUIDs which are groups this user is a transitive member of.
	TransitiveMemberOf collections.UUIDSet `json:"transitiveMemberOf"`
	// Discovered indicates that this user was newly discovered. This does not
	// necessarily imply the user was recently added in Azure Active Directory,
	// but it does indicate that it's the first time the user has been seen by
	// the input.
	Discovered bool `json:"-"`
	// Modified indicates that an attribute or group membership has been
	// modified on this user.
	Modified bool `json:"-"`
	// Deleted indicates the user has been deleted.
	Deleted bool `json:"deleted"`
	// MFA contains MFA registration details for this user. This field is not
	// persisted; it is populated during each sync/update cycle when the "mfa"
	// enrich_with option is set.
	MFA *MFARegistrationDetails `json:"-"`
}

// MFARegistrationDetails contains MFA registration information for a user
// retrieved from the /reports/authenticationMethods/userRegistrationDetails
// endpoint. This data is not persisted across sync cycles.
type MFARegistrationDetails struct {
	IsMFACapable                                  bool     `json:"isMfaCapable"`
	IsMFARegistered                               bool     `json:"isMfaRegistered"`
	IsPasswordlessCapable                         bool     `json:"isPasswordlessCapable"`
	IsSsprCapable                                 bool     `json:"isSsprCapable"`
	IsSsprEnabled                                 bool     `json:"isSsprEnabled"`
	IsSsprRegistered                              bool     `json:"isSsprRegistered"`
	IsSystemPreferredAuthenticationMethodEnabled  bool     `json:"isSystemPreferredAuthenticationMethodEnabled"`
	MethodsRegistered                             []string `json:"methodsRegistered"`
	SystemPreferredAuthenticationMethods          []string `json:"systemPreferredAuthenticationMethods"`
	UserPreferredMethodForSecondaryAuthentication string   `json:"userPreferredMethodForSecondaryAuthentication"`
	UserType                                      string   `json:"userType"`
}

// Merge will merge the attributes and group memberships of another User
// instance into this User. The IDs of both users must match.
func (u *User) Merge(other *User) {
	if u.ID != other.ID {
		return
	}
	for k, v := range other.Fields {
		u.Fields[k] = v
	}
	other.MemberOf.ForEach(func(elem uuid.UUID) {
		u.MemberOf.Add(elem)
	})
	other.TransitiveMemberOf.ForEach(func(elem uuid.UUID) {
		u.TransitiveMemberOf.Add(elem)
	})
	u.Deleted = other.Deleted
}
