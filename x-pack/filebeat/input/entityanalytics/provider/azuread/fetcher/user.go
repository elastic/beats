// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fetcher

import (
	"github.com/google/uuid"

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
	MemberOf *collections.Set[uuid.UUID] `json:"memberOf"`
	// A set of UUIDs which are groups this user is a transitive member of.
	TransitiveMemberOf *collections.Set[uuid.UUID] `json:"transitiveMemberOf"`
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
		u.AddMemberOf(elem)
	})
	other.TransitiveMemberOf.ForEach(func(elem uuid.UUID) {
		u.AddTransitiveMemberOf(elem)
	})
	u.Deleted = other.Deleted
}

// IsMemberOf returns true if this user is a member of a group with the given ID.
func (u *User) IsMemberOf(value uuid.UUID) bool {
	if u.MemberOf != nil {
		return u.MemberOf.Has(value)
	}

	return false
}

// AddMemberOf adds the group ID to the user's MemberOf set.
func (u *User) AddMemberOf(value uuid.UUID) {
	if u.MemberOf == nil {
		u.MemberOf = collections.NewSet[uuid.UUID](value)
	} else {
		u.MemberOf.Add(value)
	}
}

// RemoveMemberOf removes the group ID from the user's MemberOf set.
func (u *User) RemoveMemberOf(value uuid.UUID) {
	if u.MemberOf != nil {
		u.MemberOf.Remove(value)
	}
}

// IsTransitiveMemberOf returns true if this user is a transitive member of a
// group with the given ID.
func (u *User) IsTransitiveMemberOf(value uuid.UUID) bool {
	if u.TransitiveMemberOf != nil {
		return u.TransitiveMemberOf.Has(value)
	}

	return false
}

// AddTransitiveMemberOf adds the group ID to the user's TransitiveMemberOf set.
func (u *User) AddTransitiveMemberOf(value uuid.UUID) {
	if u.TransitiveMemberOf == nil {
		u.TransitiveMemberOf = collections.NewSet[uuid.UUID](value)
	} else {
		u.TransitiveMemberOf.Add(value)
	}
}

// RemoveTransitiveMemberOf removes the group ID from the user's TransitiveMemberOf set.
func (u *User) RemoveTransitiveMemberOf(value uuid.UUID) {
	if u.TransitiveMemberOf != nil {
		u.TransitiveMemberOf.Remove(value)
	}
}
