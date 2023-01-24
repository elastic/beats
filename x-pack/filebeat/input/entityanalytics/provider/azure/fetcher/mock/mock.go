// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mock

import (
	"context"

	"github.com/google/uuid"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azure/fetcher"
	"github.com/elastic/elastic-agent-libs/logp"
)

var groupResponse = []*fetcher.Group{
	{
		ID:   uuid.MustParse("331676df-b8fd-4492-82ed-02b927f8dd80"),
		Name: "group1",
		Members: []fetcher.Member{
			{
				ID:   uuid.MustParse("5ebc6a0f-05b7-4f42-9c8a-682bbc75d0fc"),
				Type: fetcher.MemberUser,
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

var userResponse = []*fetcher.User{
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

type mock struct {
}

func (f *mock) Groups(ctx context.Context, _ string) ([]*fetcher.Group, string, error) {
	return groupResponse, "", nil
}

func (f *mock) Users(ctx context.Context, _ string) ([]*fetcher.User, string, error) {
	return userResponse, "", nil
}

func (f *mock) SetLogger(logger *logp.Logger) {}

func New() fetcher.Fetcher {
	return &mock{}
}
