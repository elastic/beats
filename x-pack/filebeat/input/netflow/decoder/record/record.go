// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package record

import (
	"time"
)

// Type is an enumeration type used to distinguish between the different
// types of records.
type Type uint8

const (
	// Flow enumeration value identifies exported flows.
	Flow Type = iota

	// Options enumeration value identifies exported options records, as defined
	// in NetFlowV9 and IPFIX.
	Options
)

// Map type is a regular map with string keys and interface{} values. The valid
// types for Map entries in a record are:
//
//  +---------+----------------------------------------+
//  | uint64  | unsigned integer fields.               |
//  +---------+----------------------------------------+
//  | int64   | signed integer fields.                 |
//  +---------+----------------------------------------+
//  | float64 | floating-point fields.                 |
//  +---------+----------------------------------------+
//  | bool    | boolean fields.                        |
//  +---------+----------------------------------------+
//  |[]byte   | octetArray (raw) fields.               |
//  +---------+----------------------------------------+
//  | string  | string fields.                         |
//  +---------+----------------------------------------+
//  |time.Time| timestamp fields.                      |
//  +---------+----------------------------------------+
//  | net.IP  | IPv4 and IPv6 address fields.          |
//  +---------+----------------------------------------+
//  |  Map    | nested fields found in option records. |
//  +---------+----------------------------------------+
type Map map[string]interface{}

// Record represents a NetFlow record extracted from a NetFlow packet.
type Record struct {
	// Time of export for this record. This timestamp is obtained from
	// the NetFlow header so its accuracy depends on the Exporter's clock.
	Timestamp time.Time

	// Fields included in this record. For static NetFlow protocols
	// (versions 1 to 8), these fields are the V9/IPFIX equivalent of
	// the original fields.
	// For NetFlow 9 and IPFIX flow records, this is a map of the fields included
	// in each flow.
	// For NetFlow 9 and IPFIX options records, this map contains two submaps,
	// one for scope and one for options.
	Fields Map

	// Exporter contains metadata from the exporter process and NetFlow session.
	// Valid keys are:
	//
	// +--------------+-----------+------------------------------------------------------------------+
	// | version      |   uint16  | The NetFlow version used to transport the record                 |
	// +--------------+-----------+------------------------------------------------------------------+
	// | timestamp    | time.Time | Publishing time at the exporter process.                         |
	// +--------------+-----------+------------------------------------------------------------------+
	// | uptimeMillis |   uint64  | Time in milliseconds that the exporter process has been running. |
	// +--------------+-----------+------------------------------------------------------------------+
	// | address      |   string  | Network address of the exporter process, in <ip>:<port> format.  |
	// +--------------+-----------+------------------------------------------------------------------+
	//
	// NetFlow 5 only:
	// +------------------+-----------+------------------------------------------------------------+
	// | samplingInterval |   uint64  | Aggregation method being used (See AggType for details).   |
	// +------------------+-----------+------------------------------------------------------------+
	//
	// NetFlow 5, 6, 8 only:
	// +--------------+-----------+------------------------------------------------------------------+
	// | engineType   |   uint64  | Type of flow-switching engine.                                   |
	// +--------------+-----------+------------------------------------------------------------------+
	// | engineId     |   uint64  | ID number of the flow switching engine.                          |
	// +--------------+-----------+------------------------------------------------------------------+
	//
	// NetFlow 8 only:
	// +--------------------+-----------+------------------------------------------------------------+
	// | aggregation        |   uint64  | Aggregation method being used (See AggType for details).   |
	// +--------------------+-----------+------------------------------------------------------------+
	// | aggregationVersion |   uint64  | Version of the aggregation export.                         |
	// +--------------------+-----------+------------------------------------------------------------+
	//
	// NetFlow 9 & IPFIX only:
	// +--------------+-----------+------------------------------------------------------------------+
	// | sourceId     |   uint64  | Exporter observation domain ID.                                  |
	// +--------------+-----------+------------------------------------------------------------------+
	Exporter Map

	// Type is the type of this record, either Flow or Options.
	Type Type
}
