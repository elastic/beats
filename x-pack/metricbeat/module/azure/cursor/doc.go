// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

// Package cursor provides persistent cursor state for the azure metricbeat module.
// It stores the last successful collection end time per metricset so that the
// module can backfill gaps after agent restarts.
package cursor
