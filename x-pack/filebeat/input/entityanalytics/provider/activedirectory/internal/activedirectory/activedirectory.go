// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package activedirectory provides Active Directory user and group query support.
package activedirectory

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
)

var (
	ErrInvalidDistinguishedName = errors.New("invalid base distinguished name")
	ErrGroups                   = errors.New("failed to get group details")
	ErrUsers                    = errors.New("failed to get user details")
)

var cnUsers = &ldap.RelativeDN{Attributes: []*ldap.AttributeTypeAndValue{{Type: "CN", Value: "Users"}}}

// Entry is an Active Directory user entry with associated group membership.
type Entry struct {
	ID          string         `json:"id"`
	User        map[string]any `json:"user"`
	Groups      []any          `json:"groups,omitempty"`
	WhenChanged time.Time      `json:"whenChanged"`
}

// GetDetails returns all the users in the Active directory with the provided base
// on the host with the given ldap url (ldap://, ldaps://, ldapi:// or cldap://).
// Group membership details are collected and added to the returned documents. If
// the group query fails, the user details query will still be attempted, but
// a non-nil error indicating the failure will be returned. If since is non-zero
// only records with whenChanged since that time will be returned. since is
// expected to be configured in a time zone the Active Directory server will
// understand, most likely UTC.
func GetDetails(url, user, pass string, base *ldap.DN, since time.Time, pagingSize uint32, dialer *net.Dialer, tlsconfig *tls.Config) ([]Entry, error) {
	if base == nil || len(base.RDNs) == 0 {
		return nil, fmt.Errorf("%w: no path", ErrInvalidDistinguishedName)
	}
	baseDN := base.String()
	if !base.RDNs[0].Equal(cnUsers) {
		return nil, fmt.Errorf("%w: %s does not have %s", ErrInvalidDistinguishedName, baseDN, cnUsers)
	}

	var opts []ldap.DialOpt
	if dialer != nil {
		opts = append(opts, ldap.DialWithDialer(dialer))
	}
	if tlsconfig != nil {
		opts = append(opts, ldap.DialWithTLSConfig(tlsconfig))
	}
	conn, err := ldap.DialURL(url, opts...)
	if err != nil {
		return nil, err
	}

	err = conn.Bind(user, pass)
	if err != nil {
		return nil, err
	}
	defer conn.Unbind()

	var errs []error

	// Format update epoch moment.
	var sinceFmtd string
	if !since.IsZero() {
		const denseTimeLayout = "20060102150405.0Z" // Differs from the const below in resolution and behaviour.
		sinceFmtd = since.Format(denseTimeLayout)
	}

	// Get groups in the directory. Get all groups independent of the
	// since parameter as they may not have changed for changed users.
	var groups directory
	grps, err := search(conn, baseDN, "(objectClass=group)", pagingSize)
	if err != nil {
		// Allow continuation if groups query fails, but warn.
		errs = []error{fmt.Errorf("%w: %w", ErrGroups, err)}
		groups.Entries = entries{}
	} else {
		groups = collate(grps, nil)
	}

	// Get users in the directory...
	userFilter := "(objectClass=user)"
	if sinceFmtd != "" {
		userFilter = "(&(objectClass=user)(whenChanged>=" + sinceFmtd + "))"
	}
	usrs, err := search(conn, baseDN, userFilter, pagingSize)
	if err != nil {
		errs = append(errs, fmt.Errorf("%w: %w", ErrUsers, err))
		return nil, errors.Join(errs...)
	}
	// ...and apply group membership.
	users := collate(usrs, groups.Entries)

	// Also collect users that are members of groups that have changed.
	if sinceFmtd != "" {
		grps, err := search(conn, baseDN, "(&(objectClass=groups)(whenChanged>="+sinceFmtd+"))", pagingSize)
		if err != nil {
			// Allow continuation if groups query fails, but warn.
			errs = append(errs, fmt.Errorf("failed to collect changed groups: %w: %w", ErrGroups, err))
		} else {
			groups := collate(grps, nil)

			// Get users of the changed groups
			var modGrps []string
			for _, e := range groups.Entries {
				dn, ok := e["distinguishedName"].(string)
				if !ok {
					continue
				}
				modGrps = append(modGrps, dn)
			}
			if len(modGrps) != 0 {
				for i, u := range modGrps {
					modGrps[i] = "(memberOf=" + u + ")"
				}
				query := "(&(objectClass=user)(|" + strings.Join(modGrps, "") + ")"
				usrs, err := search(conn, baseDN, query, pagingSize)
				if err != nil {
					errs = append(errs, fmt.Errorf("failed to collect users of changed groups%w: %w", ErrUsers, err))
				} else {
					// ...and apply group membership, inserting into users
					// if not present.
					for dn, u := range collate(usrs, groups.Entries).Entries {
						_, ok := users.Entries[dn]
						if ok {
							continue
						}
						users.Entries[dn] = u
					}
				}
			}
		}
	}

	// Assemble into a set of documents.
	docs := make([]Entry, 0, len(users.Entries))
	for id, u := range users.Entries {
		user := u["user"].(map[string]any)
		var groups []any
		switch g := u["groups"].(type) {
		case nil:
		case []any:
			// Do not bother concretising these.
			groups = g
		}
		docs = append(docs, Entry{ID: id, User: user, Groups: groups, WhenChanged: whenChanged(user, groups)})
	}
	return docs, errors.Join(errs...)
}

func whenChanged(user map[string]any, groups []any) time.Time {
	l, _ := user["whenChanged"].(time.Time)
	for _, g := range groups {
		g, ok := g.(map[string]any)
		if !ok {
			continue
		}
		gl, ok := g["whenChanged"].(time.Time)
		if !ok {
			continue
		}
		if gl.After(l) {
			l = gl
		}
	}
	return l
}

// search performs an LDAP filter search on conn at the LDAP base. If paging
// is non-zero, page sizing will be used. See [ldap.Conn.SearchWithPaging] for
// details.
func search(conn *ldap.Conn, base, filter string, pagingSize uint32) (*ldap.SearchResult, error) {
	srch := &ldap.SearchRequest{
		BaseDN:       base,
		Scope:        ldap.ScopeWholeSubtree,
		DerefAliases: ldap.NeverDerefAliases,
		SizeLimit:    0,
		TimeLimit:    0,
		TypesOnly:    false,
		Filter:       filter,
		Attributes:   nil,
		Controls:     nil,
	}
	if pagingSize != 0 {
		return conn.SearchWithPaging(srch, pagingSize)
	}
	return conn.Search(srch)
}

// entries is a set of LDAP entries keyed on the entities distinguished name
// and then the name of the attribute.
type entries map[string]map[string]any

type directory struct {
	Entries   entries  `json:"entries"`
	Referrals []string `json:"referrals"`
	Controls  []string `json:"controls"`
}

// collate renders an LDAP search result in to a map[string]any, annotating with
// group information if it is available. Fields with known types will be converted
// from strings to the known type.
// Also included in the returned map is the sets of referrals and controls.
func collate(resp *ldap.SearchResult, groups entries) directory {
	dir := directory{
		Entries: make(entries),
	}
	for _, e := range resp.Entries {
		u := make(map[string]any)
		m := u
		if groups != nil {
			m = map[string]any{"user": u}
		}
		for _, attr := range e.Attributes {
			val := entype(attr)
			u[attr.Name] = val
			if groups != nil && attr.Name == "memberOf" {
				switch val := val.(type) {
				case []string:
					if len(val) != 0 {
						grps := make([]any, 0, len(val))
						for _, n := range val {
							g, ok := groups[n]
							if !ok {
								continue
							}
							grps = append(grps, g)
						}
						if len(grps) != 0 {
							m["groups"] = grps
						}
					}

				case string:
					g, ok := groups[val]
					if ok {
						m["groups"] = []any{g}
					}
				}
			}
		}
		dir.Entries[e.DN] = m
	}

	// Do we want this information? If not, remove this stanza. If we
	// do, we should include the information in the Entries returned
	// by the exposed API.
	if len(resp.Referrals) != 0 {
		dir.Referrals = resp.Referrals
	}
	if len(resp.Controls) != 0 {
		dir.Controls = make([]string, 0, len(resp.Controls))
		for _, e := range resp.Controls {
			if e == nil {
				continue
			}
			dir.Controls = append(dir.Controls, e.String())
		}
	}

	return dir
}

// entype converts LDAP attributes with known types to their known type if
// possible, falling back to the string if not.
func entype(attr *ldap.EntryAttribute) any {
	if len(attr.Values) == 0 {
		return attr.Values
	}
	switch attr.Name {
	case "isCriticalSystemObject", "showInAdvancedViewOnly":
		if len(attr.Values) != 1 {
			return attr.Values
		}
		switch {
		case strings.EqualFold(attr.Values[0], "true"):
			return true
		case strings.EqualFold(attr.Values[0], "false"):
			return false
		default:
			return attr.Values[0]
		}
	case "whenCreated", "whenChanged", "dSCorePropagationData":
		var times []time.Time
		if len(attr.Values) > 1 {
			times = make([]time.Time, 0, len(attr.Values))
		}
		for _, v := range attr.Values {
			const denseTimeLayout = "20060102150405.999999999Z"
			t, err := time.Parse(denseTimeLayout, v)
			if err != nil {
				return attr.Values
			}
			if len(attr.Values) == 1 {
				return t
			}
			times = append(times, t)
		}
		return times
	case "accountExpires", "lastLogon", "lastLogonTimestamp", "pwdLastSet":
		var times []time.Time
		if len(attr.Values) > 1 {
			times = make([]time.Time, 0, len(attr.Values))
		}
		for _, v := range attr.Values {
			ts, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return attr.Values
			}
			if len(attr.Values) == 1 {
				return fromWindowsNT(ts)
			}
			times = append(times, fromWindowsNT(ts))
		}
		return times
	case "objectGUID", "objectSid":
		if len(attr.ByteValues) == 1 {
			return attr.ByteValues[0]
		}
		return attr.ByteValues
	}
	if len(attr.Values) == 1 {
		return attr.Values[0]
	}
	return attr.Values
}

// epochDelta is the unix epoch in ldap time.
const epochDelta = 116444736000000000

var unixEpoch = time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)

func fromWindowsNT(ts int64) time.Time {
	return unixEpoch.Add(time.Duration(ts-epochDelta) * 100)
}
