package wmi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewDefaultConfig verifies the default values for the Config struct.
func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	assert.False(t, cfg.IncludeQueries, "IncludeQueries should default to false")
	assert.False(t, cfg.IncludeNull, "IncludeNull should default to false")
	assert.Equal(t, WMIDefaultNamespace, cfg.Namespace, "Namespace should default to WMIDefaultNamespace")
	assert.Empty(t, cfg.Queries, "Queries should default to an empty slice")
}

// TestValidateConnectionParameters checks the validation logic for user and password.
func TestValidateConnectionParameters(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		expectedError string
	}{
		{
			name:          "Valid user and password",
			config:        Config{User: "admin", Password: "password"},
			expectedError: "",
		},
		{
			name:          "User without password",
			config:        Config{User: "admin"},
			expectedError: "if user is set, password should be set",
		},
		{
			name:          "Password without user",
			config:        Config{Password: "password"},
			expectedError: "if password is set, user should be set",
		},
		{
			name:          "No user and no password",
			config:        Config{},
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateConnectionParameters()
			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.expectedError)
			}
		})
	}
}

// TestCompileQueries ensures queries are properly compiled.
func TestCompileQueries(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		expectedError string
	}{
		{
			name: "Valid queries",
			config: Config{
				Queries: []QueryConfig{
					{
						Class:  "Win32_Process",
						Fields: []string{"Name", "ID"},
						Where:  "Name LIKE 'chrome%'",
					},
				},
			},
			expectedError: "",
		},
		{
			name:          "No queries defined",
			config:        Config{},
			expectedError: "at least a query is needed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.CompileQueries()
			if tt.expectedError == "" {
				assert.NoError(t, err)
				assert.NotEmpty(t, tt.config.Queries[0].QueryStr, "QueryStr should be compiled and not empty")
			} else {
				assert.EqualError(t, err, tt.expectedError)
			}
		})
	}
}

// TestQueryCompile ensures individual query compilation works correctly.
func TestQueryCompile(t *testing.T) {
	tests := []struct {
		name          string
		queryConfig   QueryConfig
		expectedQuery string
	}{
		{
			name: "Simple query with WHERE clause",
			queryConfig: QueryConfig{
				Class:  "Win32_Process",
				Fields: []string{"Name", "ProcessId"},
				Where:  "Name = 'notepad.exe'",
			},
			expectedQuery: "SELECT Name,ProcessId FROM Win32_Process WHERE Name = 'notepad.exe'",
		},
		{
			name: "Query with multiple fields and no WHERE clause",
			queryConfig: QueryConfig{
				Class:  "Win32_Service",
				Fields: []string{"Name", "State", "StartMode"},
				Where:  "",
			},
			expectedQuery: "SELECT Name,State,StartMode FROM Win32_Service",
		},
		{
			name: "Query with wildcard (*) for fields",
			queryConfig: QueryConfig{
				Class:  "Win32_ComputerSystem",
				Fields: []string{},
				Where:  "Manufacturer = 'Dell'",
			},
			expectedQuery: "SELECT * FROM Win32_ComputerSystem WHERE Manufacturer = 'Dell'",
		},
		{
			name: "Query with wildcard (*) and no WHERE clause",
			queryConfig: QueryConfig{
				Class:  "Win32_BIOS",
				Fields: []string{"*"},
				Where:  "",
			},
			expectedQuery: "SELECT * FROM Win32_BIOS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.queryConfig.compileQuery()
			assert.Equal(t, tt.expectedQuery, tt.queryConfig.QueryStr, "QueryStr should match the expected query string")
		})
	}
}
