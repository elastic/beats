// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

// generate the application id map
//go:generate go run ./generate

package jumplists

import (
	"context"
	"errors"
	"math"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/client"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/filters"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/interfaces"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	jumpliststypes "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/jumplists"
	elasticjumplists "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/jumplists/elastic_jumplists"
)

func init() {
	elasticjumplists.RegisterGenerateFunc(getResults)
}

type jumplistType string

const (
	jumplistTypeCustom    jumplistType = "custom"
	jumplistTypeAutomatic jumplistType = "automatic"
)

// jumplistMeta is metadata shared by every entry from one jump list file.
type Meta struct {
	*jumpliststypes.ApplicationID
	*jumpliststypes.UserProfile
	*jumpliststypes.JumplistMeta
}

// Entry is a single entry in a jump list.
type Entry struct {
	*DestListEntry
	*Lnk
}

// jumplist holds entries from one jump list source file.
type jumplist struct {
	*Meta
	entries []*Entry
}

// jumplistRow is one emitted row.
type jumplistRow struct {
	*Meta
	*Entry
}

// toRows converts a jump list to row objects.
func (j *jumplist) toRows() []jumplistRow {
	var rows []jumplistRow
	for _, entry := range j.entries {
		rows = append(rows, jumplistRow{
			Meta:  j.Meta,
			Entry: entry,
		})
	}
	return rows
}

// matchesFilters is a helper function that checks if a row matches the given filters.
func matchesFilters(row jumplistRow, filters []filters.Filter, log *logger.Logger) bool {
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
func getAllJumplists(log *logger.Logger, client ClientInterface) ([]*jumplist, error) {
	var jumplists []*jumplist

	userProfiles, err := getUserProfiles(log, client)
	if err != nil {
		return nil, err
	}
	for _, userProfile := range userProfiles {
		jumplists = append(jumplists, userProfile.getJumplists(log)...)
	}

	return jumplists, nil
}

func getResults(_ context.Context, queryContext table.QueryContext, log *logger.Logger, resilientClient *client.ResilientClient) ([]elasticjumplists.Result, error) {
	if resilientClient == nil {
		return nil, errors.New("jumplists client is not configured")
	}

	jumplists, err := getAllJumplists(log, resilientClient)
	if err != nil {
		return nil, err
	}

	var results []elasticjumplists.Result
	constraintFilters := filters.GetConstraintFilters(queryContext)
	for _, jumpList := range jumplists {
		for _, row := range jumpList.toRows() {
			if matchesFilters(row, constraintFilters, log) {
				results = append(results, jumplistRowToResult(row))
			}
		}
	}
	return results, nil
}

func jumplistRowToResult(row jumplistRow) elasticjumplists.Result {
	result := elasticjumplists.Result{}

	result.ApplicationID = row.ApplicationID
	result.UserProfile = row.UserProfile
	result.JumplistMeta = row.JumplistMeta

	if row.DestListEntry != nil {
		result.DestListEntry = &jumpliststypes.DestListEntry{
			Hostname:              row.Hostname,
			EntryNumber:           row.EntryNumber,
			LastModifiedTime:      row.LastModifiedTime,
			IsPinned:              row.PinStatus,
			InteractionCount:      row.InteractionCount,
			DestEntryPath:         row.Path,
			DestEntryPathResolved: row.ResolvedPath,
			MacAddress:            row.MacAddress,
			CreationTime:          row.CreationTime,
		}
	}

	if row.Lnk != nil {
		var fileSize int32
		var volumeLabelOffset int32

		if row.FileSize > uint32(math.MaxInt32) {
			fileSize = math.MaxInt32
		} else {
			fileSize = int32(row.FileSize) //nolint:gosec,G115 // This is already safety checked in the code above
		}

		if row.VolumeLabelOffset > uint32(math.MaxInt32) {
			volumeLabelOffset = math.MaxInt32
		} else {
			volumeLabelOffset = int32(row.VolumeLabelOffset) //nolint:gosec,G115 // This is already safety checked in the code above
		}

		result.LnkMetadata = &jumpliststypes.LnkMetadata{
			LocalPath:              row.LocalPath,
			FileSize:               fileSize,
			HotKey:                 row.HotKey,
			IconIndex:              row.IconIndex,
			ShowWindow:             row.ShowWindow,
			IconLocation:           row.IconLocation,
			CommandLineArguments:   row.CommandLineArguments,
			TargetModificationTime: row.TargetModificationDate,
			TargetLastAccessedTime: row.TargetLastAccessedDate,
			TargetCreationTime:     row.TargetCreationDate,
			VolumeSerialNumber:     row.VolumeSerialNumber,
			VolumeType:             row.VolumeType,
			VolumeLabel:            row.VolumeLabel,
			VolumeLabelOffset:      volumeLabelOffset,
			Name:                   row.Name,
		}
	}

	return result
}
