// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package mock provides a mock fetcher for testing purposes.
package mock

import (
	"context"

	"github.com/google/uuid"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azuread/fetcher"
	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	GroupDeltaLinkResponse  = "group-delta-link"
	UserDeltaLinkResponse   = "user-delta-link"
	DeviceDeltaLinkResponse = "device-delta-link"
)

var GroupResponse = []*fetcher.Group{
	{
		ID:   uuid.MustParse("331676df-b8fd-4492-82ed-02b927f8dd80"),
		Name: "group1",
		Members: []fetcher.Member{
			{
				ID:   uuid.MustParse("5ebc6a0f-05b7-4f42-9c8a-682bbc75d0fc"),
				Type: fetcher.MemberUser,
			},
			{
				ID:   uuid.MustParse("6a59ea83-02bd-468f-a40b-f2c3d1821983"),
				Type: fetcher.MemberDevice,
			},
		},
	},
	{
		ID:   uuid.MustParse("d140978f-d641-4f01-802f-4ecc1acf8935"),
		Name: "group2",
		Members: []fetcher.Member{
			{
				ID:   uuid.MustParse("331676df-b8fd-4492-82ed-02b927f8dd80"),
				Type: fetcher.MemberGroup,
			},
			{
				ID:   uuid.MustParse("d897d560-3d17-4dae-81b3-c898fe82bf84"),
				Type: fetcher.MemberUser,
			},
			{
				ID:   uuid.MustParse("adbbe40a-0627-4328-89f1-88cac84dbc7f"),
				Type: fetcher.MemberDevice,
			},
		},
	},
	{
		ID:   uuid.MustParse("10db9800-3908-40cc-81c5-511fa8ccf7fd"),
		Name: "group3",
		Members: []fetcher.Member{
			{
				ID:   uuid.MustParse("d140978f-d641-4f01-802f-4ecc1acf8935"),
				Type: fetcher.MemberGroup,
			},
		},
	},
}

var UserResponse = []*fetcher.User{
	{
		ID: uuid.MustParse("5ebc6a0f-05b7-4f42-9c8a-682bbc75d0fc"),
		Fields: map[string]interface{}{
			"userPrincipalName": "user.one@example.com",
			"mail":              "user.one@example.com",
			"displayName":       "User One",
			"givenName":         "User",
			"surname":           "One",
			"jobTitle":          "Software Engineer",
			"mobilePhone":       "123-555-1000",
			"businessPhones":    []string{"123-555-0122"},
		},
	},
	{
		ID: uuid.MustParse("d897d560-3d17-4dae-81b3-c898fe82bf84"),
		Fields: map[string]interface{}{
			"userPrincipalName": "user.two@example.com",
			"mail":              "user.two@example.com",
			"displayName":       "User Two",
			"givenName":         "User",
			"surname":           "Two",
			"jobTitle":          "Accountant",
			"mobilePhone":       "205-555-2000",
			"businessPhones":    []string{"205-555-5488", "205-555-7724"},
		},
	},
}

var DeviceResponse = []*fetcher.Device{
	{
		ID: uuid.MustParse("6a59ea83-02bd-468f-a40b-f2c3d1821983"),
		Fields: map[string]interface{}{
			"accountEnabled":         true,
			"deviceId":               "eab73519-780d-4d43-be6d-a4a89af2a348",
			"displayName":            "DESKTOP-LK3PESR",
			"operatingSystem":        "Windows",
			"operatingSystemVersion": "10.0.19043.1237",
			"physicalIds":            []interface{}{},
			"extensionAttributes": map[string]interface{}{
				"extensionAttribute1": "BYOD-Device",
				"extensionAttribute2": nil,
				"extensionAttribute3": nil,
				"extensionAttribute4": nil,
			},
			"alternativeSecurityIds": []interface{}{
				map[string]interface{}{
					"type":             "2", // Rendered as string to avoid in-flight conversion to float.
					"identityProvider": nil,
					"key":              "WAA1ADAAOQA6AD...QBnAD0A",
				},
			},
		},
	},
	{
		ID: uuid.MustParse("adbbe40a-0627-4328-89f1-88cac84dbc7f"),
		Fields: map[string]interface{}{
			"accountEnabled":         true,
			"deviceId":               "2fbbb8f9-ff67-4a21-b867-a344d18a4198",
			"displayName":            "DESKTOP-LETW452G",
			"operatingSystem":        "Windows",
			"operatingSystemVersion": "10.0.19043.1337",
			"physicalIds":            []interface{}{},
			"extensionAttributes": map[string]interface{}{
				"extensionAttribute1": "BYOD-Device",
				"extensionAttribute2": nil,
				"extensionAttribute3": nil,
				"extensionAttribute4": nil,
			},
			"alternativeSecurityIds": []interface{}{
				map[string]interface{}{
					"type":             "2", // Rendered as string to avoid in-flight conversion to float.
					"identityProvider": nil,
					"key":              "DGFSGHSGGTH345A...35DSFH0A",
				},
			},
		},
	},
}

type mock struct{}

// Groups returns a fixed set of groups.
func (f *mock) Groups(ctx context.Context, _ string) ([]*fetcher.Group, string, error) {
	return GroupResponse, GroupDeltaLinkResponse, nil
}

// Users returns a fixed set of users.
func (f *mock) Users(ctx context.Context, _ string) ([]*fetcher.User, string, error) {
	return UserResponse, UserDeltaLinkResponse, nil
}

// Devices returns a fixed set of devices.
func (f *mock) Devices(ctx context.Context, _ string) ([]*fetcher.Device, string, error) {
	return DeviceResponse, DeviceDeltaLinkResponse, nil
}

// SetLogger is not used for this implementation.
func (f *mock) SetLogger(logger *logp.Logger) {}

// New creates a new instance of a mock fetcher.
func New() fetcher.Fetcher {
	return &mock{}
}
