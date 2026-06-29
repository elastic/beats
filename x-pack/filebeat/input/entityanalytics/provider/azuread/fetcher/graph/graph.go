// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package graph provides a fetcher implementation for Microsoft's Graph API,
// which is used for retrieving user and group identity assets from Azure
// Active Directory.
package graph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/gofrs/uuid/v5"
	"go.elastic.co/ecszap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/collections"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azuread/authenticator"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azuread/fetcher"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/internal/httplog"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/paths"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

const (
	defaultAPIEndpoint = "https://graph.microsoft.com/v1.0"

	queryName           = "$select"
	defaultGroupsQuery  = "displayName,members"
	defaultUsersQuery   = "accountEnabled,userPrincipalName,mail,displayName,givenName,surname,jobTitle,officeLocation,mobilePhone,businessPhones"
	defaultDevicesQuery = "accountEnabled,deviceId,displayName,operatingSystem,operatingSystemVersion,physicalIds,extensionAttributes,alternativeSecurityIds"
	expandName          = "$expand"

	apiGroupType  = "#microsoft.graph.group"
	apiUserType   = "#microsoft.graph.user"
	apiDeviceType = "#microsoft.graph.device"
<<<<<<< HEAD
=======

	mfaDetailsPath     = "/reports/authenticationMethods/userRegistrationDetails"
	signInActivityPath = "/users"
>>>>>>> f5afcc48c (x-pack/filebeat/entityanalytics/azuread: add sign-in activity enrichment for users (#51390))
)

// apiUserResponse matches the format of a user response from the Graph API.
type apiUserResponse struct {
	NextLink  string    `json:"@odata.nextLink"`
	DeltaLink string    `json:"@odata.deltaLink"`
	Users     []userAPI `json:"value"`
}

// apiGroupResponse matches the format of a group response from the Graph API.
type apiGroupResponse struct {
	NextLink  string     `json:"@odata.nextLink"`
	DeltaLink string     `json:"@odata.deltaLink"`
	Groups    []groupAPI `json:"value"`
}

// apiDeviceResponse matches the format of a user response from the Graph API.
type apiDeviceResponse struct {
	NextLink  string      `json:"@odata.nextLink"`
	DeltaLink string      `json:"@odata.deltaLink"`
	Devices   []deviceAPI `json:"value"`
}

<<<<<<< HEAD
=======
// apiMFAResponse matches the format of a userRegistrationDetails response from the Graph API.
type apiMFAResponse struct {
	NextLink string       `json:"@odata.nextLink"`
	Details  []mfaDetails `json:"value"`
}

// apiSignInActivityResponse matches the format of a sign-in activity response from the Graph API.
type apiSignInActivityResponse struct {
	NextLink string              `json:"@odata.nextLink"`
	Users    []signInActivityAPI `json:"value"`
}

// signInActivityAPI matches the format of a single user entry in a sign-in activity response.
type signInActivityAPI struct {
	ID             string                 `json:"id"`
	SignInActivity *signInActivityDetails `json:"signInActivity"`
}

// signInActivityDetails matches the format of the signInActivity object from the API.
type signInActivityDetails struct {
	LastSignInDateTime                string `json:"lastSignInDateTime"`
	LastSignInRequestId               string `json:"lastSignInRequestId"`
	LastNonInteractiveSignInDateTime  string `json:"lastNonInteractiveSignInDateTime"`
	LastNonInteractiveSignInRequestId string `json:"lastNonInteractiveSignInRequestId"`
	LastSuccessfulSignInDateTime      string `json:"lastSuccessfulSignInDateTime"`
	LastSuccessfulSignInRequestId     string `json:"lastSuccessfulSignInRequestId"`
}

// mfaDetails matches the format of a single userRegistrationDetails entry from the API.
type mfaDetails struct {
	ID                                            string   `json:"id"`
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

>>>>>>> f5afcc48c (x-pack/filebeat/entityanalytics/azuread: add sign-in activity enrichment for users (#51390))
// userAPI matches the format of user data from the API.
type userAPI mapstr.M

// groupAPI matches the format of group data from the API.
type groupAPI struct {
	ID           uuid.UUID   `json:"id"`
	DisplayName  string      `json:"displayName"`
	MembersDelta []memberAPI `json:"members@delta,omitempty"`
	Removed      *removed    `json:"@removed,omitempty"`
}

// deleted returns true if the group has been marked as deleted.
func (g *groupAPI) deleted() bool {
	return g.Removed != nil
}

// deviceAPI matches the format of device data from the API.
type deviceAPI mapstr.M

// memberAPI matches the format of group member data from the API.
type memberAPI struct {
	ID      uuid.UUID `json:"id"`
	Type    string    `json:"@odata.type"`
	Removed *removed  `json:"@removed,omitempty"`
}

// deleted returns true if the group member has been marked as deleted.
func (o *memberAPI) deleted() bool {
	return o.Removed != nil
}

// removed matches the format of the @removed field from the API.
type removed struct {
	Reason string `json:"reason"`
}

// conf contains parameters needed to configure the fetcher.
type graphConf struct {
	APIEndpoint string    `config:"api_endpoint"`
	Select      selection `config:"select"`
	Expand      expansion `config:"expand"`

	Transport httpcommon.HTTPTransportSettings `config:",inline"`

	// Tracer allows configuration of request trace logging.
	Tracer *tracerConfig `config:"tracer"`
}

type tracerConfig struct {
	Enabled           *bool `config:"enabled"`
	lumberjack.Logger `config:",inline"`
}

// This is required due to circularity.
const inputName = "azure-ad"

func (c *tracerConfig) Validate() error {
	if !c.enabled() {
		return nil
	}
	if c.Filename == "" {
		return errors.New("request tracer must have a filename if used")
	}
	if c.MaxSize == 0 {
		// By default Lumberjack caps file sizes at 100MB which
		// is excessive for a debugging logger, so default to 1MB
		// which is the minimum.
		c.MaxSize = 1
	}
	return nil
}

func (c *tracerConfig) enabled() bool {
	return c != nil && (c.Enabled == nil || *c.Enabled)
}

type selection struct {
	UserQuery   []string `config:"users"`
	GroupQuery  []string `config:"groups"`
	DeviceQuery []string `config:"devices"`
}

type expansion struct {
	UserExpansion   map[string][]string `config:"users"`
	GroupExpansion  map[string][]string `config:"groups"`
	DeviceExpansion map[string][]string `config:"devices"`
}

// graph implements the fetcher.Fetcher interface.
type graph struct {
	conf   graphConf
	client *http.Client
	logger *logp.Logger
	auth   authenticator.Authenticator

	usersURL           string
	groupsURL          string
	devicesURL         string
	deviceOwnerUserURL string
<<<<<<< HEAD
=======
	mfaDetailsURL      string
	signInActivityURL  string
>>>>>>> f5afcc48c (x-pack/filebeat/entityanalytics/azuread: add sign-in activity enrichment for users (#51390))
}

// SetLogger sets the logger on this fetcher.
func (f *graph) SetLogger(logger *logp.Logger) {
	f.logger = logger
}

// Groups retrieves group identity assets from Azure Active Directory using
// Microsoft's Graph API. If a delta link is given, it will be used to resume
// from the last query, and only changed groups will be returned. Otherwise,
// a full list of known groups will be returned. In either case, a new delta link
// will be returned as well.
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
			return groups, "", nextLinkLoopError{"groups"}
		}
		if response.NextLink != "" {
			fetchURL = response.NextLink
		} else {
			return groups, "", missingLinkError{"groups"}
		}
	}
}

// Users retrieves user identity assets from Azure Active Directory using
// Microsoft's Graph API. If a delta link is given, it will be used to resume
// from the last query, and only changed users will be returned. Otherwise,
// a full list of known users will be returned. In either case, a new delta link
// will be returned as well.
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
				f.logger.Errorw("Unable to parse user from API", "error", err)
				continue
			}
			f.logger.Debugf("Got user %q from API", user.ID)
			users = append(users, user)
		}

		if response.DeltaLink != "" {
			return users, response.DeltaLink, nil
		}
		if response.NextLink == fetchURL {
			return users, "", nextLinkLoopError{"users"}
		}
		if response.NextLink != "" {
			fetchURL = response.NextLink
		} else {
			return users, "", missingLinkError{"users"}
		}
	}
}

// Devices retrieves device identity assets from Azure Active Directory using
// Microsoft's Graph API. If a delta link is given, it will be used to resume
// from the last query, and only changed users will be returned. Otherwise,
// a full list of known users will be returned. In either case, a new delta link
// will be returned as well.
func (f *graph) Devices(ctx context.Context, deltaLink string) ([]*fetcher.Device, string, error) {
	var devices []*fetcher.Device

	fetchURL := f.devicesURL
	if deltaLink != "" {
		fetchURL = deltaLink
	}

	for {
		var response apiDeviceResponse

		body, err := f.doRequest(ctx, http.MethodGet, fetchURL, nil)
		if err != nil {
			return nil, "", fmt.Errorf("unable to fetch devices: %w", err)
		}

		dec := json.NewDecoder(body)
		if err = dec.Decode(&response); err != nil {
			_ = body.Close()
			return nil, "", fmt.Errorf("unable to decode devices response: %w", err)
		}
		_ = body.Close()

		for _, v := range response.Devices {
			device, err := newDeviceFromAPI(v)
			if err != nil {
				f.logger.Errorw("Unable to parse device from API", "error", err)
				continue
			}
			f.logger.Debugf("Got device %q from API", device.ID)

			f.addRegistered(ctx, device, "registeredOwners", &device.RegisteredOwners)
			f.addRegistered(ctx, device, "registeredUsers", &device.RegisteredUsers)

			devices = append(devices, device)
		}

		if response.DeltaLink != "" {
			return devices, response.DeltaLink, nil
		}
		if response.NextLink == fetchURL {
			return devices, "", nextLinkLoopError{"devices"}
		}
		if response.NextLink != "" {
			fetchURL = response.NextLink
		} else {
			return devices, "", missingLinkError{"devices"}
		}
	}
}

// addRegistered adds registered owner or user UUIDs to the provided device.
func (f *graph) addRegistered(ctx context.Context, device *fetcher.Device, typ string, set *collections.UUIDSet) {
	usersLink := fmt.Sprintf("%s/%s/%s", f.deviceOwnerUserURL, device.ID, typ) // ID here is the object ID.
	users, _, err := f.Users(ctx, usersLink)
	switch {
	case err == nil, errors.Is(err, nextLinkLoopError{"users"}), errors.Is(err, missingLinkError{"users"}):
	default:
		f.logger.Errorw("Failed to obtain some registered user data", "error", err)
	}
	for _, u := range users {
		set.Add(u.ID)
	}
}

<<<<<<< HEAD
=======
// UserMFADetails retrieves MFA registration details for all users from Azure
// Active Directory using Microsoft's Graph API. Returns a map from user UUID
// to MFARegistrationDetails, or an error if a failure occurred.
func (f *graph) UserMFADetails(ctx context.Context) (map[uuid.UUID]*fetcher.MFARegistrationDetails, error) {
	result := make(map[uuid.UUID]*fetcher.MFARegistrationDetails)
	fetchURL := f.mfaDetailsURL

	for {
		var response apiMFAResponse

		body, err := f.doRequest(ctx, http.MethodGet, fetchURL, nil)
		if err != nil {
			return nil, fmt.Errorf("unable to fetch MFA registration details: %w", err)
		}

		dec := json.NewDecoder(body)
		if err = dec.Decode(&response); err != nil {
			_ = body.Close()
			return nil, fmt.Errorf("unable to decode MFA registration details response: %w", err)
		}
		_ = body.Close()

		for _, d := range response.Details {
			id, err := uuid.FromString(d.ID)
			if err != nil {
				f.logger.Warnf("Skipping MFA entry with invalid user ID %q: %v", d.ID, err)
				continue
			}
			result[id] = &fetcher.MFARegistrationDetails{
				IsMFACapable:          d.IsMFACapable,
				IsMFARegistered:       d.IsMFARegistered,
				IsPasswordlessCapable: d.IsPasswordlessCapable,
				IsSsprCapable:         d.IsSsprCapable,
				IsSsprEnabled:         d.IsSsprEnabled,
				IsSsprRegistered:      d.IsSsprRegistered,
				IsSystemPreferredAuthenticationMethodEnabled: d.IsSystemPreferredAuthenticationMethodEnabled,
				MethodsRegistered:                             d.MethodsRegistered,
				SystemPreferredAuthenticationMethods:          d.SystemPreferredAuthenticationMethods,
				UserPreferredMethodForSecondaryAuthentication: d.UserPreferredMethodForSecondaryAuthentication,
				UserType: d.UserType,
			}
			f.logger.Debugf("Got MFA registration details for user %q from API", id)
		}

		if response.NextLink == "" {
			return result, nil
		}
		if response.NextLink == fetchURL {
			return result, nextLinkLoopError{"mfa_registration_details"}
		}
		fetchURL = response.NextLink
	}
}

// UserSignInActivity retrieves sign-in activity for all users from Azure
// Active Directory using Microsoft's Graph API. Returns a map from user UUID
// to SignInActivityDetails, or an error if a failure occurred.
func (f *graph) UserSignInActivity(ctx context.Context) (map[uuid.UUID]*fetcher.SignInActivityDetails, error) {
	result := make(map[uuid.UUID]*fetcher.SignInActivityDetails)
	fetchURL := f.signInActivityURL

	for {
		nextLink, err := f.fetchSignInActivityPage(ctx, fetchURL, result)
		if err != nil {
			return nil, err
		}
		if nextLink == "" {
			return result, nil
		}
		if nextLink == fetchURL {
			return result, nextLinkLoopError{"sign_in_activity"}
		}
		fetchURL = nextLink
	}
}

// fetchSignInActivityPage fetches one page of sign-in activity results and
// merges them into result. It returns the next page URL, or an empty string
// when all pages have been consumed.
func (f *graph) fetchSignInActivityPage(ctx context.Context, fetchURL string, result map[uuid.UUID]*fetcher.SignInActivityDetails) (string, error) {
	body, err := f.doRequest(ctx, http.MethodGet, fetchURL, nil)
	if err != nil {
		return "", fmt.Errorf("unable to fetch sign-in activity: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, body)
		_ = body.Close()
	}()

	var response apiSignInActivityResponse
	if err = json.NewDecoder(body).Decode(&response); err != nil {
		return "", fmt.Errorf("unable to decode sign-in activity response: %w", err)
	}

	for _, u := range response.Users {
		if u.SignInActivity == nil {
			continue
		}
		id, err := uuid.FromString(u.ID)
		if err != nil {
			f.logger.Warnf("Skipping sign-in activity entry with invalid user ID %q: %v", u.ID, err)
			continue
		}
		result[id] = &fetcher.SignInActivityDetails{
			LastSignInDateTime:                u.SignInActivity.LastSignInDateTime,
			LastSignInRequestId:               u.SignInActivity.LastSignInRequestId,
			LastNonInteractiveSignInDateTime:  u.SignInActivity.LastNonInteractiveSignInDateTime,
			LastNonInteractiveSignInRequestId: u.SignInActivity.LastNonInteractiveSignInRequestId,
			LastSuccessfulSignInDateTime:      u.SignInActivity.LastSuccessfulSignInDateTime,
			LastSuccessfulSignInRequestId:     u.SignInActivity.LastSuccessfulSignInRequestId,
		}
		f.logger.Debugf("Got sign-in activity for user %q from API", id)
	}

	return response.NextLink, nil
}

>>>>>>> f5afcc48c (x-pack/filebeat/entityanalytics/azuread: add sign-in activity enrichment for users (#51390))
// doRequest is a convenience function for making HTTP requests to the Graph API.
// It will automatically handle requesting a token using the authenticator attached
// to this fetcher.
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

// New creates a new instance of the graph fetcher.
func New(ctx context.Context, id string, cfg *config.C, logger *logp.Logger, auth authenticator.Authenticator, p *paths.Path) (fetcher.Fetcher, error) {
	var c graphConf
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("unable to unpack Graph API Fetcher config: %w", err)
	}

	if c.Tracer != nil {
		resolved, err := httplog.ResolveTraceFilename(p, inputName, id, c.Tracer.Filename)
		if err != nil {
			return nil, err
		}
		c.Tracer.Filename = resolved
	}

	client, err := c.Transport.Client(httpcommon.WithLogger(logger))
	if err != nil {
		return nil, fmt.Errorf("unable to create HTTP client: %w", err)
	}
	client = requestTrace(ctx, client, c, logger)

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
	groupsURL.RawQuery, err = formatQuery(queryName, c.Select.GroupQuery, defaultGroupsQuery, c.Expand.GroupExpansion)
	if err != nil {
		return nil, fmt.Errorf("failed to format group query: %w", err)
	}
	f.groupsURL = groupsURL.String()

	usersURL, err := url.Parse(f.conf.APIEndpoint + "/users/delta")
	if err != nil {
		return nil, fmt.Errorf("invalid users URL endpoint: %w", err)
	}
	usersURL.RawQuery, err = formatQuery(queryName, c.Select.UserQuery, defaultUsersQuery, c.Expand.UserExpansion)
	if err != nil {
		return nil, fmt.Errorf("failed to format user query: %w", err)
	}
	f.usersURL = usersURL.String()

	devicesURL, err := url.Parse(f.conf.APIEndpoint + "/devices/delta")
	if err != nil {
		return nil, fmt.Errorf("invalid devices URL endpoint: %w", err)
	}
	devicesURL.RawQuery, err = formatQuery(queryName, c.Select.DeviceQuery, defaultDevicesQuery, c.Expand.DeviceExpansion)
	if err != nil {
		return nil, fmt.Errorf("failed to format device query: %w", err)
	}
	f.devicesURL = devicesURL.String()

	// The API takes a departure from the query approach here, so we
	// need to construct a partial URL for use later when fetching
	// registered owners and users.
	ownerUserURL, err := url.Parse(f.conf.APIEndpoint + "/devices/")
	if err != nil {
		return nil, fmt.Errorf("invalid device owner/user URL endpoint: %w", err)
	}
	f.deviceOwnerUserURL = ownerUserURL.String()

<<<<<<< HEAD
=======
	mfaDetailsURL, err := url.Parse(f.conf.APIEndpoint + mfaDetailsPath)
	if err != nil {
		return nil, fmt.Errorf("invalid MFA details URL endpoint: %w", err)
	}
	f.mfaDetailsURL = mfaDetailsURL.String()

	signInActivityURL, err := url.Parse(f.conf.APIEndpoint + signInActivityPath)
	if err != nil {
		return nil, fmt.Errorf("invalid sign-in activity URL endpoint: %w", err)
	}
	signInActivityURL.RawQuery = "$select=id,signInActivity"
	f.signInActivityURL = signInActivityURL.String()

>>>>>>> f5afcc48c (x-pack/filebeat/entityanalytics/azuread: add sign-in activity enrichment for users (#51390))
	return &f, nil
}

// requestTrace decorates cli with an httplog.LoggingRoundTripper if cfg.Tracer
// is non-nil.
func requestTrace(ctx context.Context, cli *http.Client, cfg graphConf, log *logp.Logger) *http.Client {
	if cfg.Tracer == nil {
		return cli
	}
	if !cfg.Tracer.enabled() {
		// We have a trace log name, but we are not enabled,
		// so remove all trace logs we own.
		httplog.CleanTraceFiles(cfg.Tracer.Filename, log)
		return cli
	}

	w := zapcore.AddSync(cfg.Tracer)
	go func() {
		// Close the logger when we are done.
		<-ctx.Done()
		cfg.Tracer.Close()
	}()
	core := ecszap.NewCore(
		ecszap.NewDefaultEncoderConfig(),
		w,
		zap.DebugLevel,
	)
	traceLogger := zap.New(core)

	maxBodyLen := max(1, cfg.Tracer.MaxSize) * 1e6 / 10 // 10% of file max
	cli.Transport = httplog.NewLoggingRoundTripper(cli.Transport, traceLogger, maxBodyLen, log)
	return cli
}

func formatQuery(name string, query []string, dflt string, expand map[string][]string) (string, error) {
	q := dflt
	if len(query) != 0 {
		q = strings.Join(query, ",")
	}
	vals := url.Values{name: []string{q}}
	if len(expand) != 0 {
		exp := make([]string, 0, len(expand))
		for k := range expand {
			exp = append(exp, k)
		}
		sort.Strings(exp)
		for i, k := range exp {
			v, err := formatQuery(name, expand[k], q, nil)
			if err != nil {
				return "", err
			}
			exp[i] = fmt.Sprintf("%s(%s)", k, v)
		}
		vals.Add(expandName, strings.Join(exp, ","))
	}
	return url.QueryUnescape(vals.Encode())
}

// newUserFromAPI translates an API-representation of a user to a fetcher.User.
func newUserFromAPI(u userAPI) (*fetcher.User, error) {
	var newUser fetcher.User
	var err error

	newUser.Fields = mapstr.M(u)

	if idRaw, ok := newUser.Fields["id"]; ok {
		idStr, _ := idRaw.(string)
		if newUser.ID, err = uuid.FromString(idStr); err != nil {
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

// newGroupFromAPI translates an API-representation of a group to a fetcher.Group.
func newGroupFromAPI(g groupAPI) *fetcher.Group {
	newGroup := fetcher.Group{
		ID:      g.ID,
		Name:    g.DisplayName,
		Deleted: g.deleted(),
	}
	for _, v := range g.MembersDelta {
		var typ fetcher.MemberType
		switch v.Type {
		default:
			continue
		case apiUserType:
			typ = fetcher.MemberUser
		case apiGroupType:
			typ = fetcher.MemberGroup
		case apiDeviceType:
			typ = fetcher.MemberDevice
		}
		newGroup.Members = append(newGroup.Members, fetcher.Member{
			ID:      v.ID,
			Type:    typ,
			Deleted: v.deleted(),
		})
	}

	return &newGroup
}

// newDeviceFromAPI translates an API-representation of a device to a fetcher.Device.
func newDeviceFromAPI(d deviceAPI) (*fetcher.Device, error) {
	var newDevice fetcher.Device
	var err error

	newDevice.Fields = mapstr.M(d)

	if idRaw, ok := newDevice.Fields["id"]; ok {
		idStr, _ := idRaw.(string)
		if newDevice.ID, err = uuid.FromString(idStr); err != nil {
			return nil, fmt.Errorf("unable to unmarshal device, invalid ID: %w", err)
		}
		delete(newDevice.Fields, "id")
	} else {
		return nil, errors.New("device missing required id field")
	}

	if _, ok := newDevice.Fields["@removed"]; ok {
		newDevice.Deleted = true
		delete(newDevice.Fields, "@removed")
	}

	return &newDevice, nil
}

type nextLinkLoopError struct {
	endpoint string
}

func (e nextLinkLoopError) Error() string {
	return fmt.Sprintf("error during fetch %s, encountered nextLink fetch infinite loop", e.endpoint)
}

type missingLinkError struct {
	endpoint string
}

func (e missingLinkError) Error() string {
	return fmt.Sprintf("error during fetch %s, encountered response without nextLink or deltaLink", e.endpoint)
}
