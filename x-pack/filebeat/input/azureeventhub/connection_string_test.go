// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
)

var (
	namespace = "mynamespace"
	keyName   = "keyName"
	secret    = "superSecret="
	hubName   = "myhub"
)

func TestNewConnectionStringProperties(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		var happyConnStr = "Endpoint=sb://" + namespace + ".servicebus.windows.net/;SharedAccessKeyName=" + keyName + ";SharedAccessKey=" + secret + ";EntityPath=" + hubName

		props, err := parseConnectionString(happyConnStr)
		require.NoError(t, err)

		require.Equal(t, ConnectionStringProperties{
			EntityPath:              &hubName,
			Endpoint:                "sb://" + namespace + ".servicebus.windows.net/",
			FullyQualifiedNamespace: namespace + ".servicebus.windows.net",
			SharedAccessKeyName:     &keyName,
			SharedAccessKey:         &secret,
			SharedAccessSignature:   nil,
			Emulator:                false,
		}, props)
	})

	t.Run("CaseIndifference", func(t *testing.T) {
		var lowerCase = "endpoint=sb://" + namespace + ".servicebus.windows.net/;SharedAccesskeyName=" + keyName + ";sharedAccessKey=" + secret + ";Entitypath=" + hubName

		props, err := parseConnectionString(lowerCase)
		require.NoError(t, err)

		require.Equal(t, ConnectionStringProperties{
			EntityPath:              &hubName,
			Endpoint:                "sb://" + namespace + ".servicebus.windows.net/",
			FullyQualifiedNamespace: namespace + ".servicebus.windows.net",
			SharedAccessKeyName:     &keyName,
			SharedAccessKey:         &secret,
			SharedAccessSignature:   nil,
		}, props)
	})

	t.Run("NoEntityPath", func(t *testing.T) {
		var noEntityPath = "Endpoint=sb://" + namespace + ".servicebus.windows.net/;SharedAccessKeyName=" + keyName + ";SharedAccessKey=" + secret

		props, err := parseConnectionString(noEntityPath)
		require.NoError(t, err)

		require.Equal(t, ConnectionStringProperties{
			EntityPath:              nil,
			Endpoint:                "sb://" + namespace + ".servicebus.windows.net/",
			FullyQualifiedNamespace: namespace + ".servicebus.windows.net",
			SharedAccessKeyName:     &keyName,
			SharedAccessKey:         &secret,
			SharedAccessSignature:   nil,
		}, props)
	})

	t.Run("EmbeddedSAS", func(t *testing.T) {
		var withEmbeddedSAS = "Endpoint=sb://" + namespace + ".servicebus.windows.net/;SharedAccessSignature=SharedAccessSignature sr=" + namespace + ".servicebus.windows.net&sig=<base64-sig>&se=<expiry>&skn=<keyname>"

		props, err := parseConnectionString(withEmbeddedSAS)
		require.NoError(t, err)

		require.Equal(t, ConnectionStringProperties{
			EntityPath:              nil,
			Endpoint:                "sb://" + namespace + ".servicebus.windows.net/",
			FullyQualifiedNamespace: namespace + ".servicebus.windows.net",
			SharedAccessKeyName:     nil,
			SharedAccessKey:         nil,
			SharedAccessSignature:   to.Ptr("SharedAccessSignature sr=" + namespace + ".servicebus.windows.net&sig=<base64-sig>&se=<expiry>&skn=<keyname>"),
		}, props)
	})

	t.Run("WithoutEndpoint", func(t *testing.T) {
		_, err := parseConnectionString("NoEndpoint=Blah")
		require.EqualError(t, err, "key \"Endpoint\" must not be empty")
	})

	t.Run("NoSASOrKeyName", func(t *testing.T) {
		_, err := parseConnectionString("Endpoint=sb://" + namespace + ".servicebus.windows.net/")
		require.EqualError(t, err, "key \"SharedAccessKeyName\" must not be empty")
	})

	t.Run("NoSASOrKeyValue", func(t *testing.T) {
		var s = "Endpoint=sb://" + namespace + ".servicebus.windows.net/;SharedAccessKeyName=" + keyName + ";EntityPath=" + hubName

		_, err := parseConnectionString(s)
		require.EqualError(t, err, "key \"SharedAccessKey\" or \"SharedAccessSignature\" cannot both be empty")
	})

	t.Run("UseDevelopmentEmulator", func(t *testing.T) {
		cs := "Endpoint=sb://localhost:6765;SharedAccessKeyName=" + keyName + ";SharedAccessKey=" + secret + ";UseDevelopmentEmulator=true"
		parsed, err := parseConnectionString(cs)
		require.NoError(t, err)
		require.True(t, parsed.Emulator)
		require.Equal(t, "sb://localhost:6765", parsed.Endpoint)

		// also allowed _without_ a port.
		cs = "Endpoint=sb://localhost;SharedAccessKeyName=" + keyName + ";SharedAccessKey=" + secret + ";UseDevelopmentEmulator=true"
		parsed, err = parseConnectionString(cs)
		require.NoError(t, err)
		require.True(t, parsed.Emulator)
		require.Equal(t, "sb://localhost", parsed.Endpoint)

		// emulator can give connection strings that have a trailing ';'
		cs = "Endpoint=sb://localhost:6765;SharedAccessKeyName=" + keyName + ";SharedAccessKey=" + secret + ";UseDevelopmentEmulator=true;"
		parsed, err = parseConnectionString(cs)
		require.NoError(t, err)
		require.True(t, parsed.Emulator)
		require.Equal(t, "sb://localhost:6765", parsed.Endpoint)

		// UseDevelopmentEmulator works for any hostname. This allows for cases where the emulator is used
		// in testing with multiple containers, where the hostname will not be localhost but development
		// will still be local.
		cs = "Endpoint=sb://myserver.com:6765;SharedAccessKeyName=" + keyName + ";SharedAccessKey=" + secret + ";UseDevelopmentEmulator=true"
		parsed, err = parseConnectionString(cs)
		require.NoError(t, err)

		// there's no reason for a person to pass False, but it's allowed.
		// If they're not using the dev emulator then there's no special behavior, it's like a normal connection string
		cs = "Endpoint=sb://localhost:6765;SharedAccessKeyName=" + keyName + ";SharedAccessKey=" + secret + ";UseDevelopmentEmulator=false"
		parsed, err = parseConnectionString(cs)
		require.NoError(t, err)
		require.False(t, parsed.Emulator)
		require.Equal(t, "sb://localhost:6765", parsed.Endpoint)
	})
}
