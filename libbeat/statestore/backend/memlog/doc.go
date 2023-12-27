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

// Package memlog implements the memlog statestore backend.
// The store provided by memlog is a in-memory key-value store
// that logs all operations to an append only log file.
// Once the log file is considered full the store executes a checkpoint
// operation. The checkpoint operation serializes all state to a data file.
//
// The memory store in memlog holds all key-value pairs in a hashtable, with
// value represented by map[string]interface{}. As the store must be 'valid'
// based on the state of the last update operations (Set, Remove), it
// guarantees that no references into data structures passed via Set are held.
// Instead structured data is serialized/deserialized into a
// map[string]interface{}. The serialized states contain only primitive types
// like intX, uintX, float, bool, string, slices, or map[string]interface{}
// itself. As a side effect this also guarantees that the internal can always
// be serialized to disk after updating the in memory representation.
//
// On disk we have a meta file, an update log file, data files, and an active
// marker file in the store directory.
//
// The meta file only contains the store version number.
//
// Normally all operations that update the store in memory state are appended
// to the update log file.
// The file stores all entries in JSON format. Each entry starts with an action
// entry, followed by an data entry.
// The action entry has the schema: `{"op": "<name>", id: <number>}`. Supporter
// operations are 'set' or 'remove'. The `id` contains a sequential counter
// that must always be increased by 1.
// The data entry for the 'set' operation has the format: `{"K": "<key>", "V": { ... }}`.
// The data entry for the 'remove' operation has the format: `{"K": "<key>"}`.
// Updates to the log file are not synced to disk. Having all updates available
// between restarts/crashes also depends on the capabilities of the operation
// system and file system. When opening the store we read up until it is
// possible, reconstructing a last known valid state the beat can continue
// from. This can lead to duplicates if the machine/filesystem has had an
// outage with state not yet fully synchronised to disk. Ordinary restarts
// should not lead to any problems.
// If any error is encountered when reading the log file, the next updates to the store
// will trigger a checkpoint operation and reset the log file.
//
// The store might contain multiple data files, but only the last data file is
// supposed to be valid. Older data files will continiously tried to be cleaned up
// on checkpoint operations.
// The data files filenames do include the change sequence number. Which allows
// us to sort them by name. The checkpoint operation of memlog, writes the full
// state into a new data file, that consists of an JSON array with all known
// key-value pairs.  Each JSON object in the array consists of the value
// object, with memlog private fields added. Private fields start with `_`. At
// the moment the only private field is `_key`, which is used to identify the
// key-value pair.
// NOTE: Creating a new file guarantees that Beats can progress when creating a
// new checkpoint file.  Some filesystems tend to block the
// delete/replace operation when the file is accessed by another process
// (e.g. common problem with AV Scanners on Windows). By creating a new
// file we circumvent this problem.  Failures in deleting old files is
// ok, and we will try to delete old data files again in the future.
//
// The active marker file is not really used by the store. It is written for
// debugging purposes and contains the filepath of the last written data file
// that is supposed to be valid.
//
// When opening the store we first validate the meta file and read the "last"
// data file into the in-memory hashtable. Older data files are ignored. The
// filename with the update sequence number is used to sort data files.
// NOTE: the active marker file is not used, as the checkpoint operation is
// supposed to be an atomic operation that is finalized once the data
// file is moved to its correct location.
//
// After loading the data file we loop over all operations in the log file.
// Operations with a smaller sequence number are ignored when iterating the log
// file. If any subsequent entries in the log file have a sequence number difference !=
// 1, we assume the log file to be corrupted and stop the loop. All processing
// continues from the last known accumulated state.
//
// When closing the store we make a last attempt at fsyncing the log file (just
// in case), close the log file and clear all in memory state.
//
// The store provided by memlog is threadsafe and uses a RWMutex. We allow only
// one active writer, but multiple concurrent readers.
package memlog
