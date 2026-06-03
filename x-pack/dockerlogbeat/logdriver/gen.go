// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// To regenerate entry.pb.go, run: mage update
// (requires protoc and protoc-gen-gogofaster: go install github.com/gogo/protobuf/protoc-gen-gogofaster@latest)

//go:generate protoc --gogofaster_out=import_path=logdriver:. entry.proto

package logdriver
