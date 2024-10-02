// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package jamf provides Jamf API support.
package jamf

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Token is a Jamf API authentication bearer token.
type Token struct {
	Token   string    `json:"token"`
	Expires time.Time `json:"expires"`
}

// IsValidFor return whether the token is valid up until a time grace in the past.
func (t Token) IsValidFor(grace time.Duration) bool {
	return t.Token != "" && !t.Expires.IsZero() && t.Expires.After(time.Now().Add(-grace))
}

func (t Token) String() string {
	if !t.IsValidFor(0) {
		return "invalid"
	}
	return "Bearer " + t.Token
}

// GetToken returns a new bearer token for the user at the provided Jamf tenant.
func GetToken(ctx context.Context, cli *http.Client, tenant, username, password string) (Token, error) {
	const endpoint = "/api/v1/auth/token"

	u := &url.URL{
		Scheme: "https",
		Host:   tenant,
		Path:   endpoint,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return Token{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(username, password)

	resp, err := cli.Do(req)
	if err != nil {
		return Token{}, err
	}
	defer resp.Body.Close()

	var body bytes.Buffer
	n, err := io.Copy(&body, resp.Body)
	if n == 0 || err != nil {
		return Token{}, err
	}

	var errorProbe struct {
		Errors any `json:"errors"`
	}
	err = json.Unmarshal(body.Bytes(), &errorProbe)
	if err != nil {
		return Token{}, err
	}
	if errorProbe.Errors != nil {
		return Token{}, recoverError(body.Bytes())
	}
	var t Token
	err = json.Unmarshal(body.Bytes(), &t)
	if err != nil {
		return Token{}, err
	}
	return t, nil
}

type Computers struct {
	TotalCount int        `json:"totalCount"`
	Results    []Computer `json:"results"`

	Errors any `json:"errors,omitempty"` // Errors is a sentinel error detection field.
}

// Computer is an Jamf computer's details.
//
// See https://developer.jamf.com/jamf-pro/reference/get_preview-computers for details.
type Computer struct {
	Location                                Location  `json:"location,omitempty"`
	Site                                    *string   `json:"site"`
	Name                                    *string   `json:"name"`
	Udid                                    *string   `json:"udid"`
	SerialNumber                            *string   `json:"serialNumber"`
	OperatingSystemVersion                  *string   `json:"operatingSystemVersion"`
	OperatingSystemBuild                    *string   `json:"operatingSystemBuild"`
	OperatingSystemSupplementalBuildVersion *string   `json:"operatingSystemSupplementalBuildVersion"`
	OperatingSystemRapidSecurityResponse    *string   `json:"operatingSystemRapidSecurityResponse"`
	MacAddress                              *string   `json:"macAddress"`
	AssetTag                                *string   `json:"assetTag"`
	ModelIdentifier                         *string   `json:"modelIdentifier"`
	MdmAccessRights                         *int      `json:"mdmAccessRights"`
	LastContactDate                         time.Time `json:"lastContactDate"`
	LastReportDate                          time.Time `json:"lastReportDate"`
	LastEnrolledDate                        time.Time `json:"lastEnrolledDate"`
	IpAddress                               *string   `json:"ipAddress"`
	ManagementId                            *string   `json:"managementId"`
	IsManaged                               *bool     `json:"isManaged"`
}

func (c Computer) Equal(o Computer) bool {
	return ptrEq(c.Udid, o.Udid) &&
		ptrEq(c.Site, o.Site) &&
		ptrEq(c.Name, o.Name) &&
		ptrEq(c.SerialNumber, o.SerialNumber) &&
		ptrEq(c.OperatingSystemVersion, o.OperatingSystemVersion) &&
		ptrEq(c.OperatingSystemBuild, o.OperatingSystemBuild) &&
		ptrEq(c.OperatingSystemSupplementalBuildVersion, o.OperatingSystemSupplementalBuildVersion) &&
		ptrEq(c.OperatingSystemRapidSecurityResponse, o.OperatingSystemRapidSecurityResponse) &&
		ptrEq(c.MacAddress, o.MacAddress) &&
		ptrEq(c.AssetTag, o.AssetTag) &&
		ptrEq(c.ModelIdentifier, o.ModelIdentifier) &&
		ptrEq(c.MdmAccessRights, o.MdmAccessRights) &&
		c.LastContactDate.Equal(o.LastContactDate) &&
		c.LastReportDate.Equal(o.LastReportDate) &&
		c.LastEnrolledDate.Equal(o.LastEnrolledDate) &&
		ptrEq(c.IpAddress, o.IpAddress) &&
		ptrEq(c.ManagementId, o.ManagementId) &&
		ptrEq(c.IsManaged, o.IsManaged) &&
		c.Location.Equal(o.Location)
}

// Location is an Jamf location's details.
type Location struct {
	Username     *string `json:"username,omitempty"`
	RealName     *string `json:"realName,omitempty"`
	EmailAddress *string `json:"emailAddress,omitempty"`
	Position     *string `json:"position,omitempty"`
	PhoneNumber  *string `json:"phoneNumber,omitempty"`
	Department   *string `json:"department,omitempty"`
	Building     *string `json:"building,omitempty"`
	Room         *string `json:"room,omitempty"`
}

func (l Location) Equal(o Location) bool {
	return ptrEq(l.Username, o.Username) &&
		ptrEq(l.RealName, o.RealName) &&
		ptrEq(l.EmailAddress, o.EmailAddress) &&
		ptrEq(l.Position, o.Position) &&
		ptrEq(l.PhoneNumber, o.PhoneNumber) &&
		ptrEq(l.Department, o.Department) &&
		ptrEq(l.Building, o.Building) &&
		ptrEq(l.Room, o.Room)
}

func ptrEq[T comparable](a, b *T) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil, b == nil:
		return false
	default:
		return *a == *b
	}
}

// User is an Jamf user's details.
//
// See https://developer.jamf.com/jamf-pro/reference/findusers for details.
type Users struct {
	Users []User `json:"users"`

	Errors any `json:"errors,omitempty"` // Errors is a sentinel error detection field.
}

// User is a Jamf user.
type User struct {
	ID   *int    `json:"id,omitempty"`
	Name *string `json:"name,omitempty"`
}

// GetComputers returns Jamf computer details using the preview computers
// API endpoint. tenant is the Jamf tenant and tok is the API token to use for
// the query.
//
// See https://developer.jamf.com/jamf-pro/reference/get_preview-computers for details.
func GetComputers(ctx context.Context, cli *http.Client, tenant string, tok Token, query url.Values) (Computers, error) {
	const endpoint = "/api/preview/computers"

	u := &url.URL{
		Scheme:   "https",
		Host:     tenant,
		Path:     endpoint,
		RawQuery: query.Encode(),
	}
	return getDetails[Computers](ctx, cli, u, tok)
}

// GetUsers returns Jamf users using the list users API endpoint. tenant is the
// Jamf user domain and key is the API token to use for the query. If user is not empty,
// details for the specific user are returned, otherwise a list of all users is returned.
// The query parameter holds queries as described in https://developer.Jamf.com/docs/reference/user-query/
// with the query syntax described at https://developer.Jamf.com/docs/reference/core-Jamf-api/#filter.
// Parts of the response may be omitted using the omit parameter.
//
// See https://developer.jamf.com/jamf-pro/reference/findusers for details.
func GetUsers(ctx context.Context, cli *http.Client, tenant string, tok Token, query url.Values) ([]User, error) {
	const endpoint = "/JSSResource/users"

	u := &url.URL{
		Scheme:   "https",
		Host:     tenant,
		Path:     endpoint,
		RawQuery: query.Encode(),
	}
	users, err := getDetails[Users](ctx, cli, u, tok)
	return users.Users, err
}

// entity is an Jamf entity analytics entity.
type entity interface {
	Computers | Users
}

// getDetails returns Jamf details using the API endpoint in u. tok is the API
// token to use for the query.
func getDetails[E entity](ctx context.Context, cli *http.Client, u *url.URL, tok Token) (E, error) {
	var r E
	if !tok.IsValidFor(0) {
		return r, fmt.Errorf("expired token: %s", tok.Expires.Format(time.RFC3339))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return r, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", tok.String())

	resp, err := cli.Do(req)
	if err != nil {
		return r, err
	}
	defer resp.Body.Close()

	var body bytes.Buffer
	n, err := io.Copy(&body, resp.Body)
	if n == 0 || err != nil {
		return r, err
	}

	err = json.Unmarshal(body.Bytes(), &r)
	if err != nil {
		return r, errors.Join(err, recoverError(body.Bytes()))
	}
	switch p := any(r).(type) {
	case Computers:
		if p.Errors != nil {
			err = recoverError(body.Bytes())
		}
		return r, err
	case Users:
		if p.Errors != nil {
			err = recoverError(body.Bytes())
		}
		return r, err
	default:
		panic("unreachable")
	}
}

func recoverError(msg []byte) error {
	var e Error
	err := json.Unmarshal(msg, &e)
	if err != nil {
		return err
	}
	if e.Status == 0 {
		return nil
	}
	return &e
}

// Error is an Jamf API error value.
type Error struct {
	Status int `json:"httpStatus,omitempty"`
	Errors []struct {
		Code        string `json:"code"`
		Description string `json:"description"`
		ID          string `json:"id"`
		Field       any    `json:"field,omitempty"`
	} `json:"errors,omitempty"`
}

func (e *Error) Error() string {
	if len(e.Errors) == 0 {
		return fmt.Sprintf("error http status: %d", e.Status)
	}
	errors := make([]string, len(e.Errors))
	for i, c := range e.Errors {
		e := fmt.Sprintf("code=%s description=%s", c.Code, c.Description)
		if c.Field != nil {
			e += fmt.Sprintf(" field=%s", c.Field)
		}
		errors[i] = e
	}
	return fmt.Sprintf("error http status: %d: %s", e.Status, strings.Join(errors, ","))
}
