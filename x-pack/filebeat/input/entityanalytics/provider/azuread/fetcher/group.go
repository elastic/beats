// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fetcher

import "github.com/google/uuid"

// MemberType indicates the type of member in a Group.
type MemberType int

const (
	// MemberUser is a user.
	MemberUser MemberType = iota + 1
	// MemberGroup is a group.
	MemberGroup
	// MemberDevice is a device.
	MemberDevice
)

// Group represents a group identity asset.
type Group struct {
	// The ID (UUIDv4) of the group.
	ID uuid.UUID `json:"id"`
	// The display name of the group.
	Name string `json:"name"`
	// Indicates the group has been deleted.
	Deleted bool `json:"deleted,omitempty"`
	// A list of members for this group.
	Members []Member `json:"-"`
}

// Member represents an identity asset that is a member of a group.
type Member struct {
	// The ID (UUIDv4) of the member.
	ID uuid.UUID
	// The type of the member.
	Type MemberType
	// Indicates the member has been deleted.
	Deleted bool
}

// GroupECS is an ECS representation of a Group.
type GroupECS struct {
	// The ID (UUIDv4) of the group, in string form.
	ID string `json:"id"`
	// The display name of the group.
	Name string `json:"name"`
}

// ToECS transforms the Group into an ECS representation.
func (g *Group) ToECS() GroupECS {
	return GroupECS{
		ID:   g.ID.String(),
		Name: g.Name,
	}
}
