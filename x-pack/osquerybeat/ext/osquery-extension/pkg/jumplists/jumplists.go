// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

// generate the application id map
//go:generate go run ./generate

package jumplists

import (
	"context"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/encoding"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/filters"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/interfaces"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

type JumplistType string

const (
	JumplistTypeCustom    JumplistType = "custom"
	JumplistTypeAutomatic JumplistType = "automatic"
)

// JumplistMeta is the metadata for a jump list.
// It contains the application ID, jump list type, path to the jump list file,
// and any jumplist type specific metadata.  The embedded fields
// have osquery tags defined in their object definitions, and our encoding package
// will automatically marshal the fields to the correct JSON format.
type JumplistMeta struct {
	*ApplicationID
	*UserProfile
	JumplistType JumplistType `osquery:"jumplist_type"`
	Path         string       `osquery:"source_file_path"`
}

// JumplistEntry is a single entry in a jump list.
// TODO: Automatic jumplists will add additional fields to the JumplistEntry object.
type JumplistEntry struct {
	*DestListEntry
	*Lnk
}

// Jumplist is a collection of Lnk objects that represent a single jump list.
// It contains the metadata for the jump list and the entries (Lnk objects).
// This is a generic object that can represent either a custom jumplist
// or an automatic jumplist. It is comprised of a JumplistMeta object and a slice of Lnk objects.
type Jumplist struct {
	*JumplistMeta
	entries []*JumplistEntry
}

// JumplistRow is a single row in a jump list.
// Each jumplist is a collection of LNK objects, but each LNK object in the jumplist
// has the same metadata (application id, jumplist type, path, etc).
// This object using embedded pointers so that multiple rows can share the same metadata.
// each embedded field has osquery tags defined in their object definitions
type JumplistRow struct {
	*JumplistMeta  // The metadata for the jump list 1Code has alerts. Press enter to view.
	*JumplistEntry // The JumplistEntry object that represents a single jump list entry
}

// ToRows converts the Jumplist to a slice of JumplistRow objects.
func (j *Jumplist) ToRows() []JumplistRow {
	var rows []JumplistRow
	for _, entry := range j.entries {
		rows = append(rows, JumplistRow{
			JumplistMeta:  j.JumplistMeta,
			JumplistEntry: entry,
		})
	}
	return rows
}

// GetColumns returns the column definitions for the JumplistRow object.
// It returns a slice of table.ColumnDefinition objects.
func GetColumns() []table.ColumnDefinition {
	columns, err := encoding.GenerateColumnDefinitions(JumplistRow{})
	if err != nil {
		return nil
	}
	return columns
}

// matchesFilters is a helper function that checks if a row matches the given filters.
func matchesFilters(row JumplistRow, filters []filters.Filter) bool {
	for _, filter := range filters {
		if !filter.Matches(row) {
			return false
		}
	}
	return true
}

type ClientInterface interface {
	interfaces.QueryExecutor
}

// getAllJumplists is a helper function that gets all the jumplists for all the user profiles.
func getAllJumplists(log *logger.Logger, client ClientInterface) ([]*Jumplist, error) {
	var jumplists []*Jumplist

	userProfiles, err := getUserProfiles(log, client)
	if err != nil {
		return nil, err
	}
	for _, userProfile := range userProfiles {
		jumplists = append(jumplists, userProfile.getJumplists(log)...)
	}

	return jumplists, nil
}

// GetGenerateFunc returns a function that can be used to generate a table of JumplistRow objects.
// It returns a function that can be used to generate a table of JumplistRow objects.
func GetGenerateFunc(log *logger.Logger, client ClientInterface) table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		jumplists, err := getAllJumplists(log, client)
		if err != nil {
			return nil, err
		}
		// Convert the jumplists to a slice of map[string]string objects that will
		var marshalledRows []map[string]string
		filters := filters.GetConstraintFilters(queryContext)
		for _, jumpList := range jumplists {
			for _, row := range jumpList.ToRows() {
				if matchesFilters(row, filters) {
					rowMap, err := encoding.MarshalToMapWithFlags(row, encoding.EncodingFlagUseNumbersZeroValues)
					if err != nil {
						return nil, err
					}
					marshalledRows = append(marshalledRows, rowMap)
				}
			}
		}
		return marshalledRows, nil
	}
}
