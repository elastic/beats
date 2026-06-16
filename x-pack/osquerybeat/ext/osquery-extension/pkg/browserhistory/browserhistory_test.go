// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/encoding"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	)

// getResultsAsMaps calls getResults and converts each Result to map[string]string for tests that assert on row["column"].
func getResultsAsMaps(ctx context.Context, queryContext table.QueryContext, log *logger.Logger) ([]map[string]string, error) {
	results, err := getResults(ctx, queryContext, log)
	if err != nil {
		return nil, err
	}
	maps := make([]map[string]string, len(results))
	for i, r := range results {
		m, err := encoding.MarshalToMap(r)
		if err != nil {
			return nil, err
		}
		maps[i] = m
	}
	return maps, nil
}

// Expected test data - these values are static and any change indicates test data has been modified
const (
	expectedTotalRows   = 49
	expectedChromeRows  = 5
	expectedEdgeRows    = 5
	expectedFirefoxRows = 5
	expectedSafariRows  = 34 // 17 for Default + 17 for profile1
)

var expectedBrowserProfiles = map[string][]string{
	"chrome":  {"Default"},
	"edge":    {"Default"},
	"firefox": {"dwpys5gk.default-release"},
	"safari":  {"Default Profile", "profile1"},
}

var expectedTimestamps = map[string]map[string][]int64{
	"chrome": {
		"Default": {1760089511, 1760089517},
	},
	"edge": {
		"Default": {1760089411, 1760089423},
	},
	"firefox": {
		"dwpys5gk.default-release": {1760089455, 1760089462},
	},
	"safari": {
		"Default Profile": {1749628922, 1749628924, 1749628925, 1749628926, 1749628931, 1749629032, 1749629099, 1749629104, 1749629105, 1749629150},
		"profile1":        {1749628922, 1749628924, 1749628925, 1749628926, 1749628931, 1749629032, 1749629099, 1749629104, 1749629105, 1749629150},
	},
}

// getTestDataDir returns the absolute path to the testdata directory
func getTestDataDir(t *testing.T) string {
	t.Helper()
	testDataDir, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatalf("Failed to get absolute path to testdata: %v", err)
	}
	return testDataDir
}

// getCurrentUserFromTestData queries the test data to get the actual user name
func getCurrentUserFromTestData(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	testDataDir := getTestDataDir(t)

	queryContext := table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"custom_data_dir": {
				Constraints: []table.Constraint{
					{Operator: table.OperatorGlob, Expression: filepath.Join(testDataDir, "*")},
				},
			},
		},
	}

	rows, err := getResultsAsMaps(ctx, queryContext, logger.New(os.Stderr, true))
	if err != nil || len(rows) == 0 {
		t.Fatalf("Failed to get user from test data: %v", err)
	}

	return rows[0]["user"]
}

// TestAllRows tests that we get the expected total number of rows
func TestAllRows(t *testing.T) {
	ctx := context.Background()
	testDataDir := getTestDataDir(t)

	queryContext := table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"custom_data_dir": {
				Constraints: []table.Constraint{
					{Operator: table.OperatorGlob, Expression: filepath.Join(testDataDir, "*")},
				},
			},
		},
	}

	rows, err := getResultsAsMaps(ctx, queryContext, logger.New(os.Stderr, true))
	if err != nil {
		t.Fatalf("GetTableRows returned error: %v", err)
	}

	if len(rows) != expectedTotalRows {
		t.Errorf("Expected %d total rows, got %d", expectedTotalRows, len(rows))
	}
}

// TestChromeFilter tests Chrome browser filtering
func TestChromeFilter(t *testing.T) {
	ctx := context.Background()
	testDataDir := getTestDataDir(t)

	queryContext := table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"custom_data_dir": {
				Constraints: []table.Constraint{
					{Operator: table.OperatorLike, Expression: filepath.Join(testDataDir, "Google%")},
				},
			},
		},
	}

	rows, err := getResultsAsMaps(ctx, queryContext, logger.New(os.Stderr, true))
	if err != nil {
		t.Fatalf("GetTableRows returned error: %v", err)
	}

	if len(rows) != expectedChromeRows {
		t.Errorf("Expected %d Chrome rows, got %d", expectedChromeRows, len(rows))
	}

	for _, row := range rows {
		browser := strings.ToLower(row["browser"])
		if !strings.Contains(browser, "chrome") && !strings.Contains(browser, "chromium") {
			t.Errorf("Expected browser to contain 'chrome' or 'chromium', got: %s", browser)
		}
	}
}

// TestEdgeFilter tests Edge browser filtering
func TestEdgeFilter(t *testing.T) {
	ctx := context.Background()
	testDataDir := getTestDataDir(t)

	queryContext := table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"custom_data_dir": {
				Constraints: []table.Constraint{
					{Operator: table.OperatorLike, Expression: filepath.Join(testDataDir, "Microsoft")},
				},
			},
		},
	}

	rows, err := getResultsAsMaps(ctx, queryContext, logger.New(os.Stderr, true))
	if err != nil {
		t.Fatalf("GetTableRows returned error: %v", err)
	}

	if len(rows) != expectedEdgeRows {
		t.Errorf("Expected %d Edge rows, got %d", expectedEdgeRows, len(rows))
	}

	for _, row := range rows {
		browser := strings.ToLower(row["browser"])
		if !strings.Contains(browser, "edge") {
			t.Errorf("Expected browser to contain 'edge', got: %s", browser)
		}
	}
}

// TestBrowserFiltering tests filtering by browser column
func TestBrowserFiltering(t *testing.T) {
	testDataDir := getTestDataDir(t)
	ctx := context.Background()

	tests := []struct {
		browser          string
		expectedCount    int
		expectedProfiles []string
	}{
		{"chrome", expectedChromeRows, expectedBrowserProfiles["chrome"]},
		{"edge", expectedEdgeRows, expectedBrowserProfiles["edge"]},
		{"firefox", expectedFirefoxRows, expectedBrowserProfiles["firefox"]},
		{"safari", expectedSafariRows, expectedBrowserProfiles["safari"]},
	}

	for _, tt := range tests {
		t.Run(tt.browser, func(t *testing.T) {
			queryContext := table.QueryContext{
				Constraints: map[string]table.ConstraintList{
					"custom_data_dir": {
						Constraints: []table.Constraint{
							{Operator: table.OperatorGlob, Expression: filepath.Join(testDataDir, "*")},
						},
					},
					"browser": {
						Constraints: []table.Constraint{
							{Operator: table.OperatorEquals, Expression: tt.browser},
						},
					},
				},
			}

			rows, err := getResultsAsMaps(ctx, queryContext, logger.New(os.Stderr, true))
			if err != nil {
				t.Fatalf("GetTableRows returned error: %v", err)
			}

			if len(rows) != tt.expectedCount {
				t.Errorf("Expected %d rows for browser %s, got %d", tt.expectedCount, tt.browser, len(rows))
			}

			// Verify all rows have the correct browser
			for _, row := range rows {
				if row["browser"] != tt.browser {
					t.Errorf("Expected browser %s, got %s", tt.browser, row["browser"])
				}
			}

			// Verify expected profiles are present
			foundProfiles := make(map[string]bool)
			for _, row := range rows {
				foundProfiles[row["profile_name"]] = true
			}
			for _, expectedProfile := range tt.expectedProfiles {
				if !foundProfiles[expectedProfile] {
					t.Errorf("Expected profile %s not found for browser %s", expectedProfile, tt.browser)
				}
			}
		})
	}
}

// TestProfileFiltering tests filtering by profile_name column
func TestProfileFiltering(t *testing.T) {
	testDataDir := getTestDataDir(t)
	ctx := context.Background()

	tests := []struct {
		profile       string
		expectedCount int
	}{
		{"Default Profile", 17},         // 17 safari Default Profile
		{"Default", 10},                 // 5 chrome + 5 edge
		{"dwpys5gk.default-release", 5}, // firefox
		{"profile1", 17},                // safari profile1
	}

	for _, tt := range tests {
		t.Run(tt.profile, func(t *testing.T) {
			queryContext := table.QueryContext{
				Constraints: map[string]table.ConstraintList{
					"custom_data_dir": {
						Constraints: []table.Constraint{
							{Operator: table.OperatorGlob, Expression: filepath.Join(testDataDir, "*")},
						},
					},
					"profile_name": {
						Constraints: []table.Constraint{
							{Operator: table.OperatorEquals, Expression: tt.profile},
						},
					},
				},
			}

			rows, err := getResultsAsMaps(ctx, queryContext, logger.New(os.Stderr, true))
			if err != nil {
				t.Fatalf("GetTableRows returned error: %v", err)
			}

			if len(rows) != tt.expectedCount {
				t.Errorf("Expected %d rows for profile %s, got %d", tt.expectedCount, tt.profile, len(rows))
			}

			// Verify all rows have the correct profile
			for _, row := range rows {
				if row["profile_name"] != tt.profile {
					t.Errorf("Expected profile_name %s, got %s", tt.profile, row["profile_name"])
				}
			}
		})
	}
}

// TestUserFiltering tests filtering by user column (dynamic user name)
func TestUserFiltering(t *testing.T) {
	testDataDir := getTestDataDir(t)
	ctx := context.Background()

	// Get the actual user name from test data (platform-dependent)
	expectedUser := getCurrentUserFromTestData(t)
	if expectedUser == "" {
		t.Skip("there is no user detected for the testdata path")
	}

	queryContext := table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"custom_data_dir": {
				Constraints: []table.Constraint{
					{Operator: table.OperatorGlob, Expression: filepath.Join(testDataDir, "*")},
				},
			},
			"user": {
				Constraints: []table.Constraint{
					{Operator: table.OperatorEquals, Expression: expectedUser},
				},
			},
		},
	}

	rows, err := getResultsAsMaps(ctx, queryContext, logger.New(os.Stderr, true))
	if err != nil {
		t.Fatalf("GetTableRows returned error: %v", err)
	}

	if len(rows) != expectedTotalRows {
		t.Errorf("Expected %d rows for user %s, got %d", expectedTotalRows, expectedUser, len(rows))
	}

	// Verify all rows have the correct user
	for _, row := range rows {
		if row["user"] != expectedUser {
			t.Errorf("Expected user %s, got %s", expectedUser, row["user"])
		}
	}
}

// TestTimestampEquals tests timestamp equality filtering for each browser/profile
func TestTimestampEquals(t *testing.T) {
	testDataDir := getTestDataDir(t)
	ctx := context.Background()

	for browser, profiles := range expectedTimestamps {
		for profile, timestamps := range profiles {
			if len(timestamps) == 0 {
				continue
			}

			t.Run(browser+"/"+profile, func(t *testing.T) {
				earliestTime := timestamps[0]

				queryContext := table.QueryContext{
					Constraints: map[string]table.ConstraintList{
						"custom_data_dir": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorGlob, Expression: filepath.Join(testDataDir, "*")},
							},
						},
						"browser": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorEquals, Expression: browser},
							},
						},
						"profile_name": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorEquals, Expression: profile},
							},
						},
						"timestamp": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorEquals, Expression: strconv.FormatInt(earliestTime, 10)},
							},
						},
					},
				}

				rows, err := getResultsAsMaps(ctx, queryContext, logger.New(os.Stderr, true))
				if err != nil {
					t.Fatalf("GetTableRows returned error: %v", err)
				}

				if len(rows) == 0 {
					t.Error("Expected at least one row for timestamp equals filter")
					return
				}

				// Verify all rows have the exact timestamp
				for _, row := range rows {
					if ts, err := strconv.ParseInt(row["timestamp"], 10, 64); err == nil {
						if ts != earliestTime {
							t.Errorf("Expected timestamp %d, got %d", earliestTime, ts)
						}
					}
				}
			})
		}
	}
}

// TestTimestampGreaterThan tests timestamp > filtering for each browser/profile
func TestTimestampGreaterThan(t *testing.T) {
	testDataDir := getTestDataDir(t)
	ctx := context.Background()

	for browser, profiles := range expectedTimestamps {
		for profile, timestamps := range profiles {
			if len(timestamps) == 0 {
				continue
			}

			t.Run(browser+"/"+profile, func(t *testing.T) {
				midIdx := len(timestamps) / 2
				midTime := timestamps[midIdx]

				queryContext := table.QueryContext{
					Constraints: map[string]table.ConstraintList{
						"custom_data_dir": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorGlob, Expression: filepath.Join(testDataDir, "*")},
							},
						},
						"browser": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorEquals, Expression: browser},
							},
						},
						"profile_name": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorEquals, Expression: profile},
							},
						},
						"timestamp": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorGreaterThan, Expression: strconv.FormatInt(midTime, 10)},
							},
						},
					},
				}

				rows, err := getResultsAsMaps(ctx, queryContext, logger.New(os.Stderr, true))
				if err != nil {
					t.Fatalf("GetTableRows returned error: %v", err)
				}

				// Verify all rows have timestamp > midTime
				for _, row := range rows {
					if ts, err := strconv.ParseInt(row["timestamp"], 10, 64); err == nil {
						if ts <= midTime {
							t.Errorf("Expected timestamp > %d, got %d", midTime, ts)
						}
					}
				}
			})
		}
	}
}

// TestTimestampLessThan tests timestamp < filtering for each browser/profile
func TestTimestampLessThan(t *testing.T) {
	testDataDir := getTestDataDir(t)
	ctx := context.Background()

	for browser, profiles := range expectedTimestamps {
		for profile, timestamps := range profiles {
			if len(timestamps) == 0 {
				continue
			}

			t.Run(browser+"/"+profile, func(t *testing.T) {
				midIdx := len(timestamps) / 2
				midTime := timestamps[midIdx]

				queryContext := table.QueryContext{
					Constraints: map[string]table.ConstraintList{
						"custom_data_dir": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorGlob, Expression: filepath.Join(testDataDir, "*")},
							},
						},
						"browser": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorEquals, Expression: browser},
							},
						},
						"profile_name": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorEquals, Expression: profile},
							},
						},
						"timestamp": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorLessThan, Expression: strconv.FormatInt(midTime, 10)},
							},
						},
					},
				}

				rows, err := getResultsAsMaps(ctx, queryContext, logger.New(os.Stderr, true))
				if err != nil {
					t.Fatalf("GetTableRows returned error: %v", err)
				}

				// Verify all rows have timestamp < midTime
				for _, row := range rows {
					if ts, err := strconv.ParseInt(row["timestamp"], 10, 64); err == nil {
						if ts >= midTime {
							t.Errorf("Expected timestamp < %d, got %d", midTime, ts)
						}
					}
				}
			})
		}
	}
}

// TestTimestampGreaterThanOrEquals tests timestamp >= filtering for each browser/profile
func TestTimestampGreaterThanOrEquals(t *testing.T) {
	testDataDir := getTestDataDir(t)
	ctx := context.Background()

	for browser, profiles := range expectedTimestamps {
		for profile, timestamps := range profiles {
			if len(timestamps) == 0 {
				continue
			}

			t.Run(browser+"/"+profile, func(t *testing.T) {
				earliestTime := timestamps[0]

				queryContext := table.QueryContext{
					Constraints: map[string]table.ConstraintList{
						"custom_data_dir": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorGlob, Expression: filepath.Join(testDataDir, "*")},
							},
						},
						"browser": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorEquals, Expression: browser},
							},
						},
						"profile_name": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorEquals, Expression: profile},
							},
						},
						"timestamp": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorGreaterThanOrEquals, Expression: strconv.FormatInt(earliestTime, 10)},
							},
						},
					},
				}

				rows, err := getResultsAsMaps(ctx, queryContext, logger.New(os.Stderr, true))
				if err != nil {
					t.Fatalf("GetTableRows returned error: %v", err)
				}

				// Verify all rows have timestamp >= earliestTime
				for _, row := range rows {
					if ts, err := strconv.ParseInt(row["timestamp"], 10, 64); err == nil {
						if ts < earliestTime {
							t.Errorf("Expected timestamp >= %d, got %d", earliestTime, ts)
						}
					}
				}
			})
		}
	}
}

// TestTimestampLessThanOrEquals tests timestamp <= filtering for each browser/profile
func TestTimestampLessThanOrEquals(t *testing.T) {
	testDataDir := getTestDataDir(t)
	ctx := context.Background()

	for browser, profiles := range expectedTimestamps {
		for profile, timestamps := range profiles {
			if len(timestamps) == 0 {
				continue
			}

			t.Run(browser+"/"+profile, func(t *testing.T) {
				latestTime := timestamps[len(timestamps)-1]

				queryContext := table.QueryContext{
					Constraints: map[string]table.ConstraintList{
						"custom_data_dir": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorGlob, Expression: filepath.Join(testDataDir, "*")},
							},
						},
						"browser": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorEquals, Expression: browser},
							},
						},
						"profile_name": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorEquals, Expression: profile},
							},
						},
						"timestamp": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorLessThanOrEquals, Expression: strconv.FormatInt(latestTime, 10)},
							},
						},
					},
				}

				rows, err := getResultsAsMaps(ctx, queryContext, logger.New(os.Stderr, true))
				if err != nil {
					t.Fatalf("GetTableRows returned error: %v", err)
				}

				// Verify all rows have timestamp <= latestTime
				for _, row := range rows {
					if ts, err := strconv.ParseInt(row["timestamp"], 10, 64); err == nil {
						if ts > latestTime {
							t.Errorf("Expected timestamp <= %d, got %d", latestTime, ts)
						}
					}
				}
			})
		}
	}
}

// TestTimestampRange tests timestamp range filtering (>= AND <=) for each browser/profile
func TestTimestampRange(t *testing.T) {
	testDataDir := getTestDataDir(t)
	ctx := context.Background()

	for browser, profiles := range expectedTimestamps {
		for profile, timestamps := range profiles {
			if len(timestamps) <= 1 {
				continue
			}

			t.Run(browser+"/"+profile, func(t *testing.T) {
				earliestTime := timestamps[0]
				latestTime := timestamps[len(timestamps)-1]

				queryContext := table.QueryContext{
					Constraints: map[string]table.ConstraintList{
						"custom_data_dir": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorGlob, Expression: filepath.Join(testDataDir, "*")},
							},
						},
						"browser": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorEquals, Expression: browser},
							},
						},
						"profile_name": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorEquals, Expression: profile},
							},
						},
						"timestamp": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorGreaterThanOrEquals, Expression: strconv.FormatInt(earliestTime, 10)},
								{Operator: table.OperatorLessThanOrEquals, Expression: strconv.FormatInt(latestTime, 10)},
							},
						},
					},
				}

				rows, err := getResultsAsMaps(ctx, queryContext, logger.New(os.Stderr, true))
				if err != nil {
					t.Fatalf("GetTableRows returned error: %v", err)
				}

				// Get unique timestamps from rows
				actualTimestamps := make(map[int64]bool)
				for _, row := range rows {
					if ts, err := strconv.ParseInt(row["timestamp"], 10, 64); err == nil {
						actualTimestamps[ts] = true
						if ts < earliestTime || ts > latestTime {
							t.Errorf("Expected timestamp in range [%d, %d], got %d", earliestTime, latestTime, ts)
						}
					}
				}

				// Verify all expected timestamps are present
				for _, expectedTS := range timestamps {
					if !actualTimestamps[expectedTS] {
						t.Errorf("Expected timestamp %d not found in results", expectedTS)
					}
				}
			})
		}
	}
}

// TestTimestampConstraints tests the timestamp constraint parsing and application
func TestTimestampConstraints(t *testing.T) {
	// Test timestamp constraint parsing
	tests := []struct {
		name           string
		constraints    map[string]table.ConstraintList
		expectedCount  int
		expectedValues []int64
	}{
		{
			name: "single equals constraint",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorEquals, Expression: "1609459200"}, // 2021-01-01 00:00:00 UTC
					},
				},
			},
			expectedCount:  1,
			expectedValues: []int64{1609459200},
		},
		{
			name: "multiple timestamp constraints",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorGreaterThan, Expression: "1609459200"},
						{Operator: table.OperatorLessThan, Expression: "1640995200"}, // 2022-01-01 00:00:00 UTC
					},
				},
			},
			expectedCount:  2,
			expectedValues: []int64{1609459200, 1640995200},
		},
		{
			name: "invalid timestamp",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorEquals, Expression: "invalid"},
						{Operator: table.OperatorEquals, Expression: "1609459200"},
					},
				},
			},
			expectedCount:  1, // Only valid timestamp should be included
			expectedValues: []int64{1609459200},
		},
		{
			name:           "no timestamp constraints",
			constraints:    map[string]table.ConstraintList{},
			expectedCount:  0,
			expectedValues: []int64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryContext := table.QueryContext{
				Constraints: tt.constraints,
			}

			constraints := getTimestampConstraints(queryContext)

			if len(constraints) != tt.expectedCount {
				t.Errorf("Expected %d constraints, got %d", tt.expectedCount, len(constraints))
			}

			for i, expected := range tt.expectedValues {
				if i >= len(constraints) {
					t.Errorf("Missing expected constraint %d with value %d", i, expected)
					continue
				}
				if constraints[i].Value != expected {
					t.Errorf("Expected constraint %d to have value %d, got %d", i, expected, constraints[i].Value)
				}
			}
		})
	}
}

// TestDatetimeFiltering tests filtering using datetime field with RFC3339 format
func TestDatetimeFiltering(t *testing.T) {
	testDataDir := getTestDataDir(t)
	ctx := context.Background()

	for browser, profiles := range expectedTimestamps {
		for profile, timestamps := range profiles {
			if len(timestamps) == 0 {
				continue
			}

			t.Run(browser+"/"+profile, func(t *testing.T) {
				// Test equals with RFC3339 format
				earliestTime := timestamps[0]
				rfc3339Time := time.Unix(earliestTime, 0).UTC().Format(time.RFC3339)

				queryContext := table.QueryContext{
					Constraints: map[string]table.ConstraintList{
						"custom_data_dir": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorGlob, Expression: filepath.Join(testDataDir, "*")},
							},
						},
						"browser": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorEquals, Expression: browser},
							},
						},
						"profile_name": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorEquals, Expression: profile},
							},
						},
						"datetime": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorEquals, Expression: rfc3339Time},
							},
						},
					},
				}

				rows, err := getResultsAsMaps(ctx, queryContext, logger.New(os.Stderr, true))
				if err != nil {
					t.Fatalf("GetTableRows returned error: %v", err)
				}

				if len(rows) == 0 {
					t.Error("Expected at least one row for datetime equals filter")
					return
				}

				// Verify datetime field is present and parseable for all rows
				for _, row := range rows {
					// Verify datetime field is present and parseable
					if dt := row["datetime"]; dt != "" {
						if _, err := time.Parse(time.RFC3339, dt); err != nil {
							t.Errorf("Invalid RFC3339 datetime in result: %s", dt)
						}
					} else {
						t.Error("Expected datetime field to be present")
					}
				}

				// Test greater than or equals with RFC3339 format (simpler boundary test)
				if len(timestamps) > 1 {
					earliestRFC3339 := time.Unix(earliestTime, 0).UTC().Format(time.RFC3339)

					queryContext.Constraints["datetime"] = table.ConstraintList{
						Constraints: []table.Constraint{
							{Operator: table.OperatorGreaterThanOrEquals, Expression: earliestRFC3339},
						},
					}

					rows, err := getResultsAsMaps(ctx, queryContext, logger.New(os.Stderr, true))
					if err != nil {
						t.Fatalf("GetTableRows returned error for >= filter: %v", err)
					}

					if len(rows) == 0 {
						t.Error("Expected at least one row for datetime >= filter")
					}

					// Verify all returned timestamps are >= earliestTime
					for _, row := range rows {
						if ts, err := strconv.ParseInt(row["timestamp"], 10, 64); err == nil {
							if ts < earliestTime {
								t.Errorf("Expected timestamp >= %d, got %d", earliestTime, ts)
							}
						}
					}
				}

				// Test less than or equals with RFC3339 format
				latestTime := timestamps[len(timestamps)-1]
				latestRFC3339 := time.Unix(latestTime, 0).UTC().Format(time.RFC3339)

				queryContext.Constraints["datetime"] = table.ConstraintList{
					Constraints: []table.Constraint{
						{Operator: table.OperatorLessThanOrEquals, Expression: latestRFC3339},
					},
				}

				rows, err = getResultsAsMaps(ctx, queryContext, logger.New(os.Stderr, true))
				if err != nil {
					t.Fatalf("GetTableRows returned error for <= filter: %v", err)
				}

				// Verify all returned timestamps are <= latestTime
				for _, row := range rows {
					if ts, err := strconv.ParseInt(row["timestamp"], 10, 64); err == nil {
						if ts > latestTime {
							t.Errorf("Expected timestamp <= %d, got %d", latestTime, ts)
						}
					}
				}
			})
		}
	}
}

// TestTimestampConversionFunctions tests the timestamp conversion functions for all browsers
func TestTimestampConversionFunctions(t *testing.T) {
	tests := []struct {
		name             string
		unixTime         int64
		expectedChromium int64
		expectedFirefox  int64
		expectedSafari   int64
	}{
		{
			name:             "zero timestamp",
			unixTime:         0,
			expectedChromium: 0,
			expectedFirefox:  0,
			expectedSafari:   0,
		},
		{
			name:             "unix epoch (1970-01-01)",
			unixTime:         0,
			expectedChromium: 0, // Special case: zero stays zero
			expectedFirefox:  0, // Special case: zero stays zero
			expectedSafari:   0, // Special case: zero stays zero
		},
		{
			name:             "year 2000",
			unixTime:         946684800,         // 2000-01-01 00:00:00 UTC
			expectedChromium: 12591158400000000, // (946684800 * 1000000) + 11644473600000000
			expectedFirefox:  946684800000000,   // 946684800 * 1000000
			expectedSafari:   -31622400,         // 946684800 - 978307200 (negative because before 2001)
		},
		{
			name:             "year 2021",
			unixTime:         1609459200,        // 2021-01-01 00:00:00 UTC
			expectedChromium: 13253932800000000, // (1609459200 * 1000000) + 11644473600000000
			expectedFirefox:  1609459200000000,  // 1609459200 * 1000000
			expectedSafari:   631152000,         // 1609459200 - 978307200
		},
		{
			name:             "year 2022",
			unixTime:         1640995200,        // 2022-01-01 00:00:00 UTC
			expectedChromium: 13285468800000000, // (1640995200 * 1000000) + 11644473600000000
			expectedFirefox:  1640995200000000,  // 1640995200 * 1000000
			expectedSafari:   662688000,         // 1640995200 - 978307200
		},
		{
			name:             "large timestamp",
			unixTime:         2147483647,        // 2038-01-19 03:14:07 UTC (32-bit max)
			expectedChromium: 13791957247000000, // (2147483647 * 1000000) + 11644473600000000
			expectedFirefox:  2147483647000000,  // 2147483647 * 1000000
			expectedSafari:   1169176447,        // 2147483647 - 978307200
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Unix to other formats
			chromiumTime := unixToChromiumTime(tt.unixTime)
			if chromiumTime != tt.expectedChromium {
				t.Errorf("unixToChromiumTime(%d) = %d, expected %d", tt.unixTime, chromiumTime, tt.expectedChromium)
			}

			firefoxTime := unixToFirefoxTime(tt.unixTime)
			if firefoxTime != tt.expectedFirefox {
				t.Errorf("unixToFirefoxTime(%d) = %d, expected %d", tt.unixTime, firefoxTime, tt.expectedFirefox)
			}

			safariTime := unixToSafariTime(tt.unixTime)
			if safariTime != tt.expectedSafari {
				t.Errorf("unixToSafariTime(%d) = %d, expected %d", tt.unixTime, safariTime, tt.expectedSafari)
			}

			// Test round-trip conversions (only for non-zero values to avoid special zero handling)
			if tt.unixTime != 0 {
				// Chromium round-trip
				roundTripChromium := chromiumTimeToUnix(chromiumTime)
				if roundTripChromium != tt.unixTime {
					t.Errorf("Chromium round-trip failed: %d -> %d -> %d", tt.unixTime, chromiumTime, roundTripChromium)
				}

				// Firefox round-trip
				roundTripFirefox := firefoxTimeToUnix(firefoxTime)
				if roundTripFirefox != tt.unixTime {
					t.Errorf("Firefox round-trip failed: %d -> %d -> %d", tt.unixTime, firefoxTime, roundTripFirefox)
				}

				// Safari round-trip
				roundTripSafari := safariTimeToUnix(float64(safariTime))
				if roundTripSafari != tt.unixTime {
					t.Errorf("Safari round-trip failed: %d -> %d -> %d", tt.unixTime, safariTime, roundTripSafari)
				}
			}
		})
	}
}

// TestTimestampConversionEdgeCases tests edge cases in timestamp conversion
func TestTimestampConversionEdgeCases(t *testing.T) {
	t.Run("chromium epoch differences", func(t *testing.T) {
		// Test that Chromium epoch difference is correctly applied
		// Difference between Jan 1, 1601 and Jan 1, 1970 should be 11644473600 seconds
		const expectedEpochDifference = int64(11644473600000000) // in microseconds

		// Unix timestamp 1 should become 1 second (1,000,000 microseconds) after Chromium epoch
		unixTime := int64(1)
		chromiumTime := unixToChromiumTime(unixTime)
		expectedChromiumTime := expectedEpochDifference + 1000000
		if chromiumTime != expectedChromiumTime {
			t.Errorf("Chromium epoch calculation error: expected %d, got %d", expectedChromiumTime, chromiumTime)
		}
	})

	t.Run("firefox microseconds precision", func(t *testing.T) {
		// Firefox uses microseconds since Unix epoch
		unixTime := int64(1609459200) // 2021-01-01 00:00:00 UTC
		firefoxTime := unixToFirefoxTime(unixTime)
		expectedFirefoxTime := unixTime * 1000000 // Convert seconds to microseconds
		if firefoxTime != expectedFirefoxTime {
			t.Errorf("Firefox microseconds conversion error: expected %d, got %d", expectedFirefoxTime, firefoxTime)
		}
	})

	t.Run("safari epoch differences", func(t *testing.T) {
		// Safari uses seconds since Jan 1, 2001 UTC (Mac OS X epoch)
		// Difference between Jan 1, 1970 and Jan 1, 2001 should be 978307200 seconds

		// Unix timestamp for 2001-01-01 00:00:00 UTC should become Safari timestamp 0
		unixTimeFor2001 := int64(978307200)
		safariTime := unixToSafariTime(unixTimeFor2001)
		if safariTime != 0 {
			t.Errorf("Safari epoch calculation error: 2001-01-01 should be 0, got %d", safariTime)
		}

		// Test that the epoch offset is correctly applied
		unixTime := unixTimeFor2001 + 1 // One second after 2001
		safariTime = unixToSafariTime(unixTime)
		if safariTime != 1 {
			t.Errorf("Safari epoch offset error: expected 1, got %d", safariTime)
		}
	})

	t.Run("negative timestamps for Safari", func(t *testing.T) {
		// Times before 2001 should result in negative Safari timestamps
		unixTime := int64(946684800) // 2000-01-01 00:00:00 UTC
		safariTime := unixToSafariTime(unixTime)
		if safariTime >= 0 {
			t.Errorf("Safari timestamp for year 2000 should be negative, got %d", safariTime)
		}
	})
}
