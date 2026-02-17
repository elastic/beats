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

// parsedBaseDN holds the result of parsing a base DN for potential group components.
type parsedBaseDN struct {
	// containerBaseDN is the container portion of the DN (OU/DC components).
	containerBaseDN string
	// potentialGroupDNs are CN components that might be groups (need validation).
	potentialGroupDNs []string
	// originalBaseDN is the original base DN string.
	originalBaseDN string
}

// parseBaseDN analyzes a distinguished name and separates potential group (CN)
// components from container components (OU, DC). CN components that appear
// before container components are extracted as potential group references
// that need to be validated against LDAP.
//
// For example, given:
//
//	CN=Admin Users,OU=Groups,DC=example,DC=com
//
// This returns:
//   - containerBaseDN: "OU=Groups,DC=example,DC=com" (the container path)
//   - potentialGroupDNs: ["CN=Admin Users,OU=Groups,DC=example,DC=com"]
//
// The potential groups must be validated with validateGroupDNs() to confirm
// they are actually groups (objectClass=group) and not containers.
func parseBaseDN(base *ldap.DN) parsedBaseDN {
	result := parsedBaseDN{}
	if base == nil || len(base.RDNs) == 0 {
		return result
	}

	result.originalBaseDN = base.String()

	// Find where container components (OU, DC) start.
	// CN components before containers are treated as potential group references.
	containerStart := -1
	for i, rdn := range base.RDNs {
		if len(rdn.Attributes) == 0 {
			continue
		}
		attrType := strings.ToUpper(rdn.Attributes[0].Type)
		if attrType == "OU" || attrType == "DC" {
			containerStart = i
			break
		}
	}

	// If no container components found, or CN components don't precede them,
	// use the base DN as-is.
	if containerStart <= 0 {
		result.containerBaseDN = result.originalBaseDN
		return result
	}

	// Extract potential group DNs (CN components before container start).
	for i := 0; i < containerStart; i++ {
		rdn := base.RDNs[i]
		if len(rdn.Attributes) == 0 {
			continue
		}
		attrType := strings.ToUpper(rdn.Attributes[0].Type)
		if attrType == "CN" {
			// Build the full DN for this potential group by including it
			// and all RDNs after it (the container path).
			groupRDNs := base.RDNs[i:]
			groupDN := &ldap.DN{RDNs: groupRDNs}
			result.potentialGroupDNs = append(result.potentialGroupDNs, groupDN.String())
		}
	}

	// Build the container base DN (starting from first OU or DC).
	containerRDNs := base.RDNs[containerStart:]
	containerBase := &ldap.DN{RDNs: containerRDNs}
	result.containerBaseDN = containerBase.String()

	return result
}

// validateGroupDNs queries LDAP to verify which of the potential group DNs
// are actually groups (objectClass=group) vs containers or other object types.
// Returns only the DNs that are confirmed to be groups.
func validateGroupDNs(conn *ldap.Conn, potentialGroupDNs []string) []string {
	if len(potentialGroupDNs) == 0 {
		return nil
	}

	var confirmedGroups []string
	for _, dn := range potentialGroupDNs {
		// Query LDAP to check if this DN is a group.
		srch := &ldap.SearchRequest{
			BaseDN:       dn,
			Scope:        ldap.ScopeBaseObject, // Only check this specific object
			DerefAliases: ldap.NeverDerefAliases,
			SizeLimit:    1,
			TimeLimit:    0,
			TypesOnly:    false,
			Filter:       "(objectClass=group)",
			Attributes:   []string{"objectClass"},
			Controls:     nil,
		}

		result, err := conn.Search(srch)
		if err != nil {
			// If the search fails (e.g., object doesn't exist), skip this DN.
			continue
		}

		// If we got a result, this DN is a group.
		if len(result.Entries) > 0 {
			confirmedGroups = append(confirmedGroups, dn)
		}
	}

	return confirmedGroups
}

// buildMemberOfFilter creates an LDAP memberOf filter from a list of group DNs.
// Returns an empty string if no groups are provided.
//
// Use the LDAP_MATCHING_RULE_IN_CHAIN matching rule (OID 1.2.840.113556.1.4.1941)
// to resolve nested membership at query time.
//
// See:
//   - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-adts/4e638665-f466-4597-93c4-12f2ebfabab5
//   - https://learn.microsoft.com/en-us/windows/win32/adsi/search-filter-syntax
//   - https://ldapwiki.com/wiki/Wiki.jsp?page=LDAP_MATCHING_RULE_IN_CHAIN
func buildMemberOfFilter(groupDNs []string) string {
	if len(groupDNs) == 0 {
		return ""
	}

	if len(groupDNs) == 1 {
		return "(memberOf:1.2.840.113556.1.4.1941:=" + ldap.EscapeFilter(groupDNs[0]) + ")"
	}

	// Multiple groups: use OR filter.
	var parts []string
	for _, dn := range groupDNs {
		parts = append(parts, "(memberOf:1.2.840.113556.1.4.1941:="+ldap.EscapeFilter(dn)+")")
	}
	return "(|" + strings.Join(parts, "") + ")"
}

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
//
// query is a complete LDAP query used to identify users, which may include
// computers, for example (&(objectCategory=person)(objectClass=user)) for human
// users or (&(objectClass=computer)(objectClass=user)) for computers. When
// since is a non-zero time.Time, the query will be conjugated with
// (whenChanged>="<SINCETIME>") into a new query.
//
// If the base DN contains group (CN) components along with container (OU/DC)
// components, the search will automatically extract the group DNs and add
// memberOf filters to find users who are members of those groups. This is
// necessary because groups are leaf objects in LDAP and don't contain users
// as children in the directory tree hierarchy.
func GetDetails(query, url, user, pass string, base *ldap.DN, since time.Time, userAttrs, grpAttrs []string, pagingSize uint32, dialer *net.Dialer, tlsconfig *tls.Config) ([]Entry, error) {
	if base == nil || len(base.RDNs) == 0 {
		return nil, fmt.Errorf("%w: no path", ErrInvalidDistinguishedName)
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

	// Parse the base DN to extract any CN components that might be groups.
	// Groups are leaf objects in LDAP, so we need to search from the
	// container base and filter by group membership.
	parsed := parseBaseDN(base)

	// Validate which CN components are actually groups (vs containers like CN=Users).
	confirmedGroups := validateGroupDNs(conn, parsed.potentialGroupDNs)

	// Determine the effective base DN and membership filter.
	var baseDN, memberOfFilter string
	if len(confirmedGroups) > 0 {
		// We have confirmed groups - search from container and filter by membership.
		baseDN = parsed.containerBaseDN
		memberOfFilter = buildMemberOfFilter(confirmedGroups)
	} else {
		// No groups found - use the original base DN as-is.
		// This handles containers like CN=Users which should use subtree search.
		baseDN = parsed.originalBaseDN
	}

	// Get groups in the directory. Get all groups independent of the
	// since parameter as they may not have changed for changed users.
	var groups directory
	grps, err := search(conn, baseDN, "(objectClass=group)", grpAttrs, pagingSize)
	if err != nil {
		// Allow continuation if groups query fails, but warn.
		errs = []error{fmt.Errorf("%w: %w", ErrGroups, err)}
		groups.Entries = entries{}
	} else {
		groups = collate(grps, nil)
	}

	// Get users in the directory...
	// Build the user filter by combining the base query with any group
	// membership filter (from CN components in base DN) and time filter.
	userFilter := query
	if memberOfFilter != "" {
		userFilter = "(&" + query + memberOfFilter + ")"
	}
	if sinceFmtd != "" {
		userFilter = "(&" + userFilter + "(whenChanged>=" + sinceFmtd + "))"
	}
	usrs, err := search(conn, baseDN, userFilter, userAttrs, pagingSize)
	if err != nil {
		errs = append(errs, fmt.Errorf("%w: %w", ErrUsers, err))
		return nil, errors.Join(errs...)
	}
	// ...and apply group membership.
	users := collate(usrs, groups.Entries)

	// Also collect users that are members of groups that have changed.
	if sinceFmtd != "" {
		grps, err := search(conn, baseDN, "(&(objectClass=groups)(whenChanged>="+sinceFmtd+"))", grpAttrs, pagingSize)
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
					// Use the LDAP_MATCHING_RULE_IN_CHAIN matching rule
					// (OID 1.2.840.113556.1.4.1941) to resolve nested
					// membership at query time.
					modGrps[i] = "(memberOf:1.2.840.113556.1.4.1941:=" + ldap.EscapeFilter(u) + ")"
				}
				changedGrpFilter := "(&" + query + "(|" + strings.Join(modGrps, "") + "))"
				// Also include the base DN membership filter if present.
				if memberOfFilter != "" {
					changedGrpFilter = "(&" + changedGrpFilter + memberOfFilter + ")"
				}
				usrs, err := search(conn, baseDN, changedGrpFilter, userAttrs, pagingSize)
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
func search(conn *ldap.Conn, base, filter string, attrs []string, pagingSize uint32) (*ldap.SearchResult, error) {
	srch := &ldap.SearchRequest{
		BaseDN:       base,
		Scope:        ldap.ScopeWholeSubtree,
		DerefAliases: ldap.NeverDerefAliases,
		SizeLimit:    0,
		TimeLimit:    0,
		TypesOnly:    false,
		Filter:       filter,
		Attributes:   attrs,
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
			// Check for special values of accountExpires.
			// See https://learn.microsoft.com/en-us/windows/win32/adschema/a-accountexpires.
			if attr.Name == "accountExpires" && (ts == 0 || ts == 0x7fff_ffff_ffff_ffff) {
				return v // Return the raw string instead of converting to time
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
