// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fetcher

import (
	"github.com/google/uuid"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/collections"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// Device represents a device identity asset.
type Device struct {
	// The ID (UUIDv4) of the device.
	ID uuid.UUID `json:"id"`
	// The attributes for the device.
	Fields mapstr.M `json:"fields"`
	// A set of UUIDs which are groups this device is a member of.
	MemberOf collections.UUIDSet `json:"memberOf"`
	// A set of UUIDs which are groups this device is a transitive member of.
	TransitiveMemberOf collections.UUIDSet `json:"transitiveMemberOf"`
	// A set of UUIDs for registered owners of this device.
	RegisteredOwners collections.UUIDSet `json:"registeredOwners"`
	// A set of UUIDs for registered users of this device.
	RegisteredUsers collections.UUIDSet `json:"registeredUsers"`
	// Discovered indicates that this device was newly discovered. This does not
	// necessarily imply the device was recently added in Azure Active Directory,
	// but it does indicate that it's the first time the device has been seen by
	// the input.
	Discovered bool `json:"-"`
	// Modified indicates that an attribute or group membership has been
	// modified on this device.
	Modified bool `json:"-"`
	// Deleted indicates the device has been deleted.
	Deleted bool `json:"deleted"`
}

// Merge will merge the attributes, owners, users and group memberships of
// another Device instance into this Device. The IDs of the devices must match.
func (d *Device) Merge(other *Device) {
	if d.ID != other.ID {
		return
	}
	for k, v := range other.Fields {
		d.Fields[k] = v
	}
	other.MemberOf.ForEach(func(elem uuid.UUID) {
		d.MemberOf.Add(elem)
	})
	other.TransitiveMemberOf.ForEach(func(elem uuid.UUID) {
		d.TransitiveMemberOf.Add(elem)
	})
	other.RegisteredOwners.ForEach(func(elem uuid.UUID) {
		d.RegisteredOwners.Add(elem)
	})
	other.RegisteredUsers.ForEach(func(elem uuid.UUID) {
		d.RegisteredUsers.Add(elem)
	})
	d.Deleted = other.Deleted
}
