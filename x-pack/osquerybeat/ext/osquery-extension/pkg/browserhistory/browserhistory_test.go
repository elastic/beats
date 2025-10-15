// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"context"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/osquery/osquery-go/plugin/table"
)

func TestGetTableRows(t *testing.T) {
	// Get the absolute path to the testdata directory
	testDataDir, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatalf("Failed to get absolute path to testdata: %v", err)
	}

	tests := []struct {
		name            string
		constraints     map[string]table.ConstraintList
		expectedRows    int
		shouldHaveError bool
	}{
		{
			name: "Glob matches all",
			constraints: map[string]table.ConstraintList{
				"custom_data_dir": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorGlob, Expression: filepath.Join(testDataDir, "*")},
					},
				},
			},
			expectedRows: 49,
		},
		{
			name: "Like matches chrome",
			constraints: map[string]table.ConstraintList{
				"custom_data_dir": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorLike, Expression: filepath.Join(testDataDir, "Google%")},
					},
				},
			},
			expectedRows: 5,
		},
		{
			name: "Equal matches edge",
			constraints: map[string]table.ConstraintList{
				"custom_data_dir": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorLike, Expression: filepath.Join(testDataDir, "Microsoft")},
					},
				},
			},
			expectedRows: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			queryContext := table.QueryContext{
				Constraints: tt.constraints,
			}

			logMessages := []string{}
			testLog := func(m string, kvs ...any) {
				logMessages = append(logMessages, m)
			}

			rows, err := GetTableRows(ctx, queryContext, testLog)

			if tt.shouldHaveError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Logf("Log messages: %v", logMessages)
				t.Fatalf("GetTableRows returned error: %v", err)
			}

			if len(rows) != tt.expectedRows {
				t.Errorf("Expected at least %d rows, got %d", tt.expectedRows, len(rows))
				t.Logf("Log messages: %v", logMessages)
				return
			}
		})
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

// TestChromiumTimestampWhere tests Chromium timestamp WHERE clause generation
func TestChromiumTimestampWhere(t *testing.T) {
	tests := []struct {
		name          string
		constraints   map[string]table.ConstraintList
		expectedWhere string
	}{
		{
			name:          "no timestamp constraints",
			constraints:   map[string]table.ConstraintList{},
			expectedWhere: "",
		},
		{
			name: "equals constraint",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorEquals, Expression: "1609459200"}, // 2021-01-01 00:00:00 UTC
					},
				},
			},
			expectedWhere: " AND (visits.visit_time >= 13253932800000000 AND visits.visit_time < 13253932801000000)", // Chromium microseconds since 1601 over one-second range
		},
		{
			name: "greater than constraint",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorGreaterThan, Expression: "1609459200"},
					},
				},
			},
			expectedWhere: " AND (visits.visit_time >= 13253932801000000)",
		},
		{
			name: "less than constraint",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorLessThan, Expression: "1609459200"},
					},
				},
			},
			expectedWhere: " AND (visits.visit_time < 13253932800000000)",
		},
		{
			name: "greater than or equals constraint",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorGreaterThanOrEquals, Expression: "1609459200"},
					},
				},
			},
			expectedWhere: " AND (visits.visit_time >= 13253932800000000)",
		},
		{
			name: "less than or equals constraint",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorLessThanOrEquals, Expression: "1609459200"},
					},
				},
			},
			expectedWhere: " AND (visits.visit_time < 13253932801000000)",
		},
		{
			name: "multiple constraints",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorGreaterThanOrEquals, Expression: "1609459200"},
						{Operator: table.OperatorLessThan, Expression: "1640995200"}, // 2022-01-01 00:00:00 UTC
					},
				},
			},
			expectedWhere: " AND (visits.visit_time >= 13253932800000000 AND visits.visit_time < 13285468800000000)",
		},
		{
			name: "zero timestamp",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorEquals, Expression: "0"},
					},
				},
			},
			expectedWhere: " AND (visits.visit_time >= 0 AND visits.visit_time < 1000000)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryContext := table.QueryContext{
				Constraints: tt.constraints,
			}

			whereClause := buildChromiumTimestampWhere(queryContext)

			if whereClause != tt.expectedWhere {
				t.Errorf("Expected WHERE clause %q, got %q", tt.expectedWhere, whereClause)
			}
		})
	}
}

// TestFirefoxTimestampWhere tests Firefox timestamp WHERE clause generation
func TestFirefoxTimestampWhere(t *testing.T) {
	tests := []struct {
		name          string
		constraints   map[string]table.ConstraintList
		expectedWhere string
	}{
		{
			name:          "no timestamp constraints",
			constraints:   map[string]table.ConstraintList{},
			expectedWhere: "",
		},
		{
			name: "equals constraint",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorEquals, Expression: "1609459200"}, // 2021-01-01 00:00:00 UTC
					},
				},
			},
			expectedWhere: " AND (hv.visit_date >= 1609459200000000 AND hv.visit_date < 1609459201000000)", // Firefox microseconds since 1970 over one-second range
		},
		{
			name: "greater than constraint",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorGreaterThan, Expression: "1609459200"},
					},
				},
			},
			expectedWhere: " AND (hv.visit_date >= 1609459201000000)",
		},
		{
			name: "less than constraint",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorLessThan, Expression: "1609459200"},
					},
				},
			},
			expectedWhere: " AND (hv.visit_date < 1609459200000000)",
		},
		{
			name: "greater than or equals constraint",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorGreaterThanOrEquals, Expression: "1609459200"},
					},
				},
			},
			expectedWhere: " AND (hv.visit_date >= 1609459200000000)",
		},
		{
			name: "less than or equals constraint",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorLessThanOrEquals, Expression: "1609459200"},
					},
				},
			},
			expectedWhere: " AND (hv.visit_date < 1609459201000000)",
		},
		{
			name: "multiple constraints",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorGreaterThanOrEquals, Expression: "1609459200"},
						{Operator: table.OperatorLessThan, Expression: "1640995200"}, // 2022-01-01 00:00:00 UTC
					},
				},
			},
			expectedWhere: " AND (hv.visit_date >= 1609459200000000 AND hv.visit_date < 1640995200000000)",
		},
		{
			name: "zero timestamp",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorEquals, Expression: "0"},
					},
				},
			},
			expectedWhere: " AND (hv.visit_date >= 0 AND hv.visit_date < 1000000)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryContext := table.QueryContext{
				Constraints: tt.constraints,
			}

			whereClause := buildFirefoxTimestampWhere(queryContext)

			if whereClause != tt.expectedWhere {
				t.Errorf("Expected WHERE clause %q, got %q", tt.expectedWhere, whereClause)
			}
		})
	}
}

// TestSafariTimestampWhere tests Safari timestamp WHERE clause generation
func TestSafariTimestampWhere(t *testing.T) {
	tests := []struct {
		name          string
		constraints   map[string]table.ConstraintList
		expectedWhere string
	}{
		{
			name:          "no timestamp constraints",
			constraints:   map[string]table.ConstraintList{},
			expectedWhere: "",
		},
		{
			name: "equals constraint",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorEquals, Expression: "1609459200"}, // 2021-01-01 00:00:00 UTC
					},
				},
			},
			expectedWhere: " AND (hv.visit_time >= 631152000 AND hv.visit_time < 631152001)", // Safari seconds since 2001 over one-second range
		},
		{
			name: "greater than constraint",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorGreaterThan, Expression: "1609459200"},
					},
				},
			},
			expectedWhere: " AND (hv.visit_time >= 631152001)",
		},
		{
			name: "less than constraint",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorLessThan, Expression: "1609459200"},
					},
				},
			},
			expectedWhere: " AND (hv.visit_time < 631152000)",
		},
		{
			name: "greater than or equals constraint",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorGreaterThanOrEquals, Expression: "1609459200"},
					},
				},
			},
			expectedWhere: " AND (hv.visit_time >= 631152000)",
		},
		{
			name: "less than or equals constraint",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorLessThanOrEquals, Expression: "1609459200"},
					},
				},
			},
			expectedWhere: " AND (hv.visit_time < 631152001)",
		},
		{
			name: "multiple constraints",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorGreaterThanOrEquals, Expression: "1609459200"},
						{Operator: table.OperatorLessThan, Expression: "1640995200"}, // 2022-01-01 00:00:00 UTC
					},
				},
			},
			expectedWhere: " AND (hv.visit_time >= 631152000 AND hv.visit_time < 662688000)",
		},
		{
			name: "zero timestamp",
			constraints: map[string]table.ConstraintList{
				"timestamp": {
					Constraints: []table.Constraint{
						{Operator: table.OperatorEquals, Expression: "0"},
					},
				},
			},
			expectedWhere: " AND (hv.visit_time >= 0 AND hv.visit_time < 1)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryContext := table.QueryContext{
				Constraints: tt.constraints,
			}

			whereClause := buildSafariTimestampWhere(queryContext)

			if whereClause != tt.expectedWhere {
				t.Errorf("Expected WHERE clause %q, got %q", tt.expectedWhere, whereClause)
			}
		})
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

// TestTimestampFiltersPerBrowser tests timestamp filters for each browser type individually
func TestTimestampFiltersPerBrowser(t *testing.T) {
	testDataDir, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatalf("Failed to get absolute path to testdata: %v", err)
	}

	ctx := context.Background()

	browserPaths := map[string]string{
		"chrome":  filepath.Join(testDataDir, "Google"),
		"edge":    filepath.Join(testDataDir, "Microsoft"),
		"firefox": filepath.Join(testDataDir, "Mozilla"),
		"safari":  filepath.Join(testDataDir, "Safari"),
	}

	for browserName, browserPath := range browserPaths {
		t.Run(browserName, func(t *testing.T) {
			// Get all data for this browser first
			queryContext := table.QueryContext{
				Constraints: map[string]table.ConstraintList{
					"custom_data_dir": {
						Constraints: []table.Constraint{
							{Operator: table.OperatorEquals, Expression: browserPath},
						},
					},
				},
			}

			logMessages := []string{}
			testLog := func(m string, kvs ...any) {
				logMessages = append(logMessages, m)
			}

			allRows, err := GetTableRows(ctx, queryContext, testLog)
			if err != nil {
				t.Fatalf("GetTableRows failed for %s: %v", browserName, err)
			}

			if len(allRows) == 0 {
				t.Skipf("No test data available for %s", browserName)
			}

			// Extract timestamps for this browser
			var browserTimestamps []int64
			timestampMap := make(map[int64]bool)

			for _, row := range allRows {
				if timestampStr, ok := row["timestamp"]; ok && timestampStr != "" {
					if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil && timestamp > 0 {
						if !timestampMap[timestamp] {
							timestampMap[timestamp] = true
							browserTimestamps = append(browserTimestamps, timestamp)
						}
					}
				}
			}

			if len(browserTimestamps) == 0 {
				t.Skipf("No valid timestamps found for %s", browserName)
			}

			// Sort timestamps
			sort.Slice(browserTimestamps, func(i, j int) bool {
				return browserTimestamps[i] < browserTimestamps[j]
			})

			earliestTime := browserTimestamps[0]
			latestTime := browserTimestamps[len(browserTimestamps)-1]

			t.Logf("%s: Found %d rows with %d unique timestamps, range: %d to %d",
				browserName, len(allRows), len(browserTimestamps), earliestTime, latestTime)

			// Test equals filter
			t.Run("equals_filter", func(t *testing.T) {
				queryContext := table.QueryContext{
					Constraints: map[string]table.ConstraintList{
						"custom_data_dir": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorEquals, Expression: browserPath},
							},
						},
						"timestamp": {
							Constraints: []table.Constraint{
								{Operator: table.OperatorEquals, Expression: strconv.FormatInt(earliestTime, 10)},
							},
						},
					},
				}

				rows, err := GetTableRows(ctx, queryContext, testLog)
				if err != nil {
					t.Fatalf("GetTableRows failed: %v", err)
				}

				// Verify all rows have the expected timestamp
				for _, row := range rows {
					if timestampStr, ok := row["timestamp"]; ok {
						if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
							if timestamp != earliestTime {
								t.Errorf("Expected timestamp %d, got %d", earliestTime, timestamp)
							}
						}
					}

					// Verify all rows are from the expected browser type
					if browser, ok := row["browser"]; ok {
						// Each browser type should have its specific identifier
						switch browserName {
						case "chrome":
							if !strings.Contains(strings.ToLower(browser), "chrome") {
								t.Logf("Chrome browser field: %s", browser) // Log for debugging, might be "chromium" or similar
							}
						case "firefox":
							if !strings.Contains(strings.ToLower(browser), "firefox") {
								t.Logf("Firefox browser field: %s", browser)
							}
						case "safari":
							if !strings.Contains(strings.ToLower(browser), "safari") {
								t.Logf("Safari browser field: %s", browser)
							}
						}
					}
				}

				if len(rows) == 0 {
					t.Errorf("Expected at least one row with timestamp %d", earliestTime)
				}
			})

			// Test range filter if we have multiple timestamps
			if len(browserTimestamps) > 1 {
				t.Run("range_filter", func(t *testing.T) {
					queryContext := table.QueryContext{
						Constraints: map[string]table.ConstraintList{
							"custom_data_dir": {
								Constraints: []table.Constraint{
									{Operator: table.OperatorEquals, Expression: browserPath},
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

					rows, err := GetTableRows(ctx, queryContext, testLog)
					if err != nil {
						t.Fatalf("GetTableRows failed: %v", err)
					}

					// Should get all the original rows back since we're using the full range
					if len(rows) != len(allRows) {
						t.Errorf("Range filter should return all %d rows, got %d", len(allRows), len(rows))
					}

					// Verify all rows are within the range
					for _, row := range rows {
						if timestampStr, ok := row["timestamp"]; ok {
							if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
								if timestamp < earliestTime || timestamp > latestTime {
									t.Errorf("Timestamp %d outside expected range [%d, %d]", timestamp, earliestTime, latestTime)
								}
							}
						}
					}
				})
			}
		})
	}
}
