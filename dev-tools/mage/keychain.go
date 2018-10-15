// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package mage

import (
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"
)

var _appleKeychain = &appleKeychain{}

type appleKeychain struct{}

// SigningIdentity represents a key pair (public/private) that can be used for
// signing.
type SigningIdentity struct {
	ID          string
	Description string
}

// ListIdentities queries the keychain to get a list of signing identities
// (certificate + private key).
func (k *appleKeychain) ListIdentities() ([]SigningIdentity, error) {
	var re = regexp.MustCompile(`(?m)^\s*\d+\)\s+(\w+)\s+"(.+)"$`)

	out, err := sh.Output("security", "find-identity", "-v")
	if err != nil {
		return nil, err
	}

	var idents []SigningIdentity
	ids := map[string]struct{}{}
	for _, match := range re.FindAllStringSubmatch(out, -1) {
		ident := SigningIdentity{ID: match[1], Description: match[2]}

		// Deduplicate
		if _, found := ids[ident.ID]; found {
			continue
		}

		idents = append(idents, ident)
		ids[ident.ID] = struct{}{}
	}

	return idents, nil
}

// AppleSigningInfo indicate if signing is enabled and specifies the identities
// to use for signing applications and installers.
type AppleSigningInfo struct {
	Sign      bool
	App       SigningIdentity
	Installer SigningIdentity
}

var (
	appleSigningInfoValue *AppleSigningInfo
	appleSigningInfoErr   error
	appleSigningInfoOnce  sync.Once
)

// GetAppleSigningInfo returns the signing identities used for code signing
// apps and installers.
//
// Environment Variables
//
// APPLE_SIGNING_ENABLED - Must be set to true to enable signing. Defaults to
//     false.
//
// APPLE_SIGNING_IDENTITY_INSTALLER - filter for selecting the signing identity
//     for installers.
//
// APPLE_SIGNING_IDENTITY_APP - filter for selecting the signing identity
//     for apps.
func GetAppleSigningInfo() (*AppleSigningInfo, error) {
	appleSigningInfoOnce.Do(func() {
		appleSigningInfoValue, appleSigningInfoErr = getAppleSigningInfo()
	})

	return appleSigningInfoValue, appleSigningInfoErr
}

func getAppleSigningInfo() (*AppleSigningInfo, error) {
	var (
		signingEnabled, _ = strconv.ParseBool(EnvOr("APPLE_SIGNING_ENABLED", "false"))
		identityInstaller = strings.ToLower(EnvOr("APPLE_SIGNING_IDENTITY_INSTALLER", "Developer ID Installer"))
		identityApp       = strings.ToLower(EnvOr("APPLE_SIGNING_IDENTITY_APP", "Developer ID Application"))
	)

	if !signingEnabled {
		return &AppleSigningInfo{Sign: false}, nil
	}

	idents, err := _appleKeychain.ListIdentities()
	if err != nil {
		return nil, err
	}

	var install, app []SigningIdentity
	for _, ident := range idents {
		id, desc := strings.ToLower(ident.ID), strings.ToLower(ident.Description)
		if strings.Contains(id, identityInstaller) || strings.Contains(desc, identityInstaller) {
			install = append(install, ident)
		}
		if strings.Contains(id, identityApp) || strings.Contains(desc, identityApp) {
			app = append(app, ident)
		}
	}

	if len(install) == 1 && len(app) == 1 {
		log.Printf("Apple Code Signing Identities:\n  App: %+v\n  Installer: %+v", app[0], install[0])
		return &AppleSigningInfo{
			Sign:      true,
			Installer: install[0],
			App:       app[0],
		}, nil
	}

	if len(install) > 1 {
		return nil, errors.Errorf("found multiple installer signing identities "+
			"that match '%v'. Set a more specific APPLE_SIGNING_IDENTITY_INSTALLER "+
			"value that will select one of %+v", identityInstaller, install)
	}

	if len(app) > 1 {
		return nil, errors.Errorf("found multiple installer signing identities "+
			"that match '%v'. Set a more specific APPLE_SIGNING_IDENTITY_APP "+
			"value that will select one of %+v", identityApp, app)
	}

	if len(install) == 0 || len(app) == 0 {
		return nil, errors.Errorf("apple signing was requested with " +
			"APPLE_SIGNING_ENABLED=true, but the required signing identities " +
			"for app and installer were not found")
	}

	return &AppleSigningInfo{Sign: false}, nil
}
