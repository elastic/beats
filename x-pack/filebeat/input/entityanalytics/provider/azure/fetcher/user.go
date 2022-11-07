// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fetcher

import (
	"github.com/google/uuid"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/collections"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type User struct {
	ID                 uuid.UUID                   `json:"id"`
	Fields             mapstr.M                    `json:"fields"`
	MemberOf           *collections.Set[uuid.UUID] `json:"memberOf"`
	TransitiveMemberOf *collections.Set[uuid.UUID] `json:"transitiveMemberOf"`
	Discovered         bool                        `json:"-"`
	Modified           bool                        `json:"-"`
	Deleted            bool                        `json:"deleted"`
}

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

func (u *User) IsMemberOf(value uuid.UUID) bool {
	if u.MemberOf != nil {
		return u.MemberOf.Has(value)
	}

	return false
}

func (u *User) AddMemberOf(value uuid.UUID) {
	if u.MemberOf == nil {
		u.MemberOf = collections.NewSet[uuid.UUID](value)
	} else {
		u.MemberOf.Add(value)
	}
}

func (u *User) RemoveMemberOf(value uuid.UUID) {
	if u.MemberOf != nil {
		u.MemberOf.Remove(value)
	}
}

func (u *User) IsTransitiveMemberOf(value uuid.UUID) bool {
	if u.TransitiveMemberOf != nil {
		return u.TransitiveMemberOf.Has(value)
	}

	return false
}

func (u *User) AddTransitiveMemberOf(value uuid.UUID) {
	if u.TransitiveMemberOf == nil {
		u.TransitiveMemberOf = collections.NewSet[uuid.UUID](value)
	} else {
		u.TransitiveMemberOf.Add(value)
	}
}

func (u *User) RemoveTransitiveMemberOf(value uuid.UUID) {
	if u.TransitiveMemberOf != nil {
		u.TransitiveMemberOf.Remove(value)
	}
}
