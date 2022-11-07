// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package graph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/google/uuid"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azure/authenticator"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azure/fetcher"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

const (
	defaultAPIEndpoint = "https://graph.microsoft.com/v1.0"

	defaultGroupsQuery = "$select=displayName,members"
	defaultUsersQuery  = "$select=accountEnabled,userPrincipalName,mail,displayName,givenName,surname,jobTitle,officeLocation,mobilePhone,businessPhones"

	apiGroupType = "#microsoft.graph.group"
	apiUserType  = "#microsoft.graph.user"
)

type apiUserResponse struct {
	NextLink  string    `json:"@odata.nextLink"`
	DeltaLink string    `json:"@odata.deltaLink"`
	Users     []userAPI `json:"value"`
}

type apiGroupResponse struct {
	NextLink  string     `json:"@odata.nextLink"`
	DeltaLink string     `json:"@odata.deltaLink"`
	Groups    []groupAPI `json:"value"`
}

type userAPI mapstr.M

type groupAPI struct {
	ID           uuid.UUID   `json:"id"`
	DisplayName  string      `json:"displayName"`
	MembersDelta []memberAPI `json:"members@delta,omitempty"`
	Removed      *removed    `json:"@removed,omitempty"`
}

func (g *groupAPI) Deleted() bool {
	return g.Removed != nil
}

type memberAPI struct {
	ID      uuid.UUID `json:"id"`
	Type    string    `json:"@odata.type"`
	Removed *removed  `json:"@removed,omitempty"`
}

func (o *memberAPI) Deleted() bool {
	return o.Removed != nil
}

type removed struct {
	Reason string `json:"reason"`
}

type graphConf struct {
	APIEndpoint string `config:"api_endpoint"`

	Transport httpcommon.HTTPTransportSettings `config:",inline"`
}

type graph struct {
	conf   graphConf
	client *http.Client
	logger *logp.Logger
	auth   authenticator.Authenticator

	usersURL  string
	groupsURL string
}

func (f *graph) SetLogger(logger *logp.Logger) {
	f.logger = logger
}

func (f *graph) Groups(ctx context.Context, deltaLink string) ([]*fetcher.Group, string, error) {
	fetchURL := f.groupsURL
	if deltaLink != "" {
		fetchURL = deltaLink
	}

	var groups []*fetcher.Group
	for {
		var response apiGroupResponse

		body, err := f.doRequest(ctx, http.MethodGet, fetchURL, nil)
		if err != nil {
			return nil, "", fmt.Errorf("unable to fetch groups: %w", err)
		}

		dec := json.NewDecoder(body)
		if err = dec.Decode(&response); err != nil {
			_ = body.Close()
			return nil, "", fmt.Errorf("unable to decode groups response: %w", err)
		}
		_ = body.Close()

		for _, v := range response.Groups {
			f.logger.Debugf("Got group %q from API", v.ID)
			groups = append(groups, newGroupFromAPI(v))
		}

		if response.DeltaLink != "" {
			return groups, response.DeltaLink, nil
		}
		if response.NextLink == fetchURL {
			return nil, "", fmt.Errorf("error during fetch groups, encountered nextLink fetch infinite loop")
		}
		if response.NextLink != "" {
			fetchURL = response.NextLink
		} else {
			return nil, "", fmt.Errorf("error during fetch groups, encountered response without nextLink or deltaLink")
		}
	}
}

func (f *graph) Users(ctx context.Context, deltaLink string) ([]*fetcher.User, string, error) {
	var users []*fetcher.User

	fetchURL := f.usersURL
	if deltaLink != "" {
		fetchURL = deltaLink
	}

	for {
		var response apiUserResponse

		body, err := f.doRequest(ctx, http.MethodGet, fetchURL, nil)
		if err != nil {
			return nil, "", fmt.Errorf("unable to fetch users: %w", err)
		}

		dec := json.NewDecoder(body)
		if err = dec.Decode(&response); err != nil {
			_ = body.Close()
			return nil, "", fmt.Errorf("unable to decode users response: %w", err)
		}
		_ = body.Close()

		for _, v := range response.Users {
			user, err := newUserFromAPI(v)
			if err != nil {
				f.logger.Errorf("Unable to parse user from API: %w", err)
				continue
			}
			f.logger.Debugf("Got user %q from API", user.ID)
			users = append(users, user)
		}

		if response.DeltaLink != "" {
			return users, response.DeltaLink, nil
		}
		if response.NextLink == fetchURL {
			return nil, "", fmt.Errorf("error during fetch users, encountered nextLink fetch infinite loop")
		}
		if response.NextLink != "" {
			fetchURL = response.NextLink
		} else {
			return nil, "", fmt.Errorf("error during fetch users, encountered response without nextLink or deltaLink")
		}
	}
}

func (f *graph) doRequest(ctx context.Context, method, url string, body io.Reader) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %w", err)
	}
	bearer, err := f.auth.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get bearer token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+bearer)

	res, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		bodyData, err := io.ReadAll(res.Body)
		_ = res.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
		}
		return nil, fmt.Errorf("unexpected status code: %d body: %s", res.StatusCode, string(bodyData))
	}

	return res.Body, nil
}

func New(cfg *config.C, logger *logp.Logger, auth authenticator.Authenticator) (fetcher.Fetcher, error) {
	var c graphConf
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("unable to unpack Graph API Fetcher config: %w", err)
	}

	client, err := c.Transport.Client()
	if err != nil {
		return nil, fmt.Errorf("unable to create HTTP client: %w", err)
	}

	f := graph{
		conf:   c,
		logger: logger,
		auth:   auth,
		client: client,
	}
	if f.conf.APIEndpoint == "" {
		f.conf.APIEndpoint = defaultAPIEndpoint
	}

	groupsURL, err := url.Parse(f.conf.APIEndpoint + "/groups/delta")
	if err != nil {
		return nil, fmt.Errorf("invalid groups URL endpoint: %w", err)
	}
	groupsURL.RawQuery = url.QueryEscape(defaultGroupsQuery)
	f.groupsURL = groupsURL.String()

	usersURL, err := url.Parse(f.conf.APIEndpoint + "/users/delta")
	if err != nil {
		return nil, fmt.Errorf("invalid groups URL endpoint: %w", err)
	}
	usersURL.RawQuery = url.QueryEscape(defaultUsersQuery)
	f.usersURL = usersURL.String()

	return &f, nil
}

func newUserFromAPI(u userAPI) (*fetcher.User, error) {
	var newUser fetcher.User
	var err error

	newUser.Fields = mapstr.M(u)

	if idRaw, ok := newUser.Fields["id"]; ok {
		idStr, _ := idRaw.(string)
		if newUser.ID, err = uuid.Parse(idStr); err != nil {
			return nil, fmt.Errorf("unable to unmarshal user, invalid ID: %w", err)
		}
		delete(newUser.Fields, "id")
	} else {
		return nil, errors.New("user missing required id field")
	}

	if _, ok := newUser.Fields["@removed"]; ok {
		newUser.Deleted = true
		delete(newUser.Fields, "@removed")
	}

	return &newUser, nil
}

func newGroupFromAPI(g groupAPI) *fetcher.Group {
	newGroup := fetcher.Group{
		ID:      g.ID,
		Name:    g.DisplayName,
		Deleted: g.Deleted(),
	}
	for _, v := range g.MembersDelta {
		if v.Type == apiUserType {
			newGroup.Members = append(newGroup.Members, fetcher.Member{
				ID:      v.ID,
				Type:    fetcher.MemberUser,
				Deleted: v.Deleted(),
			})
		} else if v.Type == apiGroupType {
			newGroup.Members = append(newGroup.Members, fetcher.Member{
				ID:      v.ID,
				Type:    fetcher.MemberGroup,
				Deleted: v.Deleted(),
			})
		}
	}

	return &newGroup
}
