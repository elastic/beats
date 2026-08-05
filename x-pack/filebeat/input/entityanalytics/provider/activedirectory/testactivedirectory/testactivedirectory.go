// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package testactivedirectory provides LDAP mocks for AD entity-analytics tests.
package testactivedirectory

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/jimlambrt/gldap"
	"github.com/stretchr/testify/require"
)

// UserDN is the distinguished name of the first user in StartLDAPServer.
const UserDN = "cn=alice,dc=example,dc=com"

// StartLDAPServer starts a gldap mock with the same fixtures as the
// activedirectory provider equivalence tests. Returns an ldap:// URL.
func StartLDAPServer(t *testing.T) string {
	t.Helper()

	s, err := gldap.NewServer()
	if err != nil {
		t.Fatalf("gldap new server: %v", err)
	}
	t.Cleanup(func() { _ = s.Stop() })

	mux, err := gldap.NewMux()
	if err != nil {
		t.Fatal(err)
	}

	err = mux.Bind(func(w *gldap.ResponseWriter, r *gldap.Request) {
		resp := r.NewBindResponse()
		resp.SetResultCode(gldap.ResultSuccess)
		_ = w.Write(resp)
	})
	if err != nil {
		t.Fatal(err)
	}

	err = mux.Search(ldapSearchHandler(t))
	if err != nil {
		t.Fatal(err)
	}

	err = mux.Unbind(func(_ *gldap.ResponseWriter, _ *gldap.Request) {})
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Router(mux); err != nil {
		t.Fatal(err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0") //nolint:noctx // only used to grab a free port
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()

	go func() { _ = s.Run(addr) }()
	require.Eventually(t, func() bool {
		return s.Ready()
	}, 10*time.Second, 20*time.Millisecond, "gldap server did not become ready")

	return "ldap://" + addr
}

func ldapSearchHandler(t *testing.T) gldap.HandlerFunc {
	t.Helper()

	users := []struct {
		dn    string
		attrs map[string][]string
	}{
		{
			dn: "cn=alice,dc=example,dc=com",
			attrs: map[string][]string{
				"cn":                {"alice"},
				"distinguishedName": {"cn=alice,dc=example,dc=com"},
				"mail":              {"alice@example.com"},
				"memberOf":          {"cn=staff,dc=example,dc=com"},
				"whenChanged":       {"20260101120000.0Z"},
			},
		},
		{
			dn: "cn=bob,dc=example,dc=com",
			attrs: map[string][]string{
				"cn":                {"bob"},
				"distinguishedName": {"cn=bob,dc=example,dc=com"},
				"mail":              {"bob@example.com"},
				"whenChanged":       {"20260101130000.0Z"},
			},
		},
	}

	devices := []struct {
		dn    string
		attrs map[string][]string
	}{
		{
			dn: "cn=workstation1,dc=example,dc=com",
			attrs: map[string][]string{
				"cn":                {"workstation1"},
				"distinguishedName": {"cn=workstation1,dc=example,dc=com"},
				"whenChanged":       {"20260101140000.0Z"},
			},
		},
	}

	return func(w *gldap.ResponseWriter, r *gldap.Request) {
		msg, err := r.GetSearchMessage()
		if err != nil {
			t.Errorf("get search message: %v", err)
			return
		}

		filter := msg.Filter

		type entry struct {
			dn    string
			attrs map[string][]string
		}
		var results []entry

		switch {
		case strings.Contains(filter, "objectClass=group"):
			// No groups in this fixture.
		case strings.Contains(filter, "objectClass=computer"):
			for _, d := range devices {
				results = append(results, entry{dn: d.dn, attrs: d.attrs})
			}
		case strings.Contains(filter, "objectCategory=person"):
			for _, u := range users {
				results = append(results, entry{dn: u.dn, attrs: u.attrs})
			}
		default:
			t.Logf("unhandled filter: %s", filter)
		}

		for _, e := range results {
			resp := r.NewSearchResponseEntry(e.dn)
			for name, vals := range e.attrs {
				resp.AddAttribute(name, vals)
			}
			_ = w.Write(resp)
		}

		done := r.NewSearchDoneResponse()
		done.SetResultCode(gldap.ResultSuccess)
		_ = w.Write(done)
	}
}
