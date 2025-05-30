// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"errors"
	"reflect"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"

	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/reader/parser"
)

// MaxWorkers, Poll, PollInterval, FileSelectors, TimeStampEpoch & ExpandEventListFromField can
// be configured at a global level, which applies to all containers. They can also be configured at individual container levels.
// Container level configurations will always override global level values.
type config struct {
	AccountName              string               `config:"account_name" validate:"required"`
	StorageURL               string               `config:"storage_url"`
	Auth                     authConfig           `config:"auth" validate:"required"`
	MaxWorkers               *int                 `config:"max_workers" validate:"max=5000"`
	Poll                     *bool                `config:"poll"`
	PollInterval             *time.Duration       `config:"poll_interval"`
	Containers               []container          `config:"containers" validate:"required"`
	FileSelectors            []fileSelectorConfig `config:"file_selectors"`
	ReaderConfig             readerConfig         `config:",inline"`
	TimeStampEpoch           *int64               `config:"timestamp_epoch"`
	ExpandEventListFromField string               `config:"expand_event_list_from_field"`
}

// container contains the config for each specific blob storage container in the root account
type container struct {
	Name                     string               `config:"name" validate:"required"`
	MaxWorkers               *int                 `config:"max_workers" validate:"max=5000"`
	Poll                     *bool                `config:"poll"`
	PollInterval             *time.Duration       `config:"poll_interval"`
	FileSelectors            []fileSelectorConfig `config:"file_selectors"`
	ReaderConfig             readerConfig         `config:",inline"`
	TimeStampEpoch           *int64               `config:"timestamp_epoch"`
	ExpandEventListFromField string               `config:"expand_event_list_from_field"`
}

// fileSelectorConfig helps filter out azure blobs based on a regex pattern
type fileSelectorConfig struct {
	Regex *match.Matcher `config:"regex" validate:"required"`
	// TODO: Add support for reader config in future
}

// readerConfig defines the options for reading the content of an azure container.
type readerConfig struct {
	Parsers  parser.Config `config:",inline"`
	Decoding decoderConfig `config:"decoding"`
}

type authConfig struct {
	SharedCredentials *sharedKeyConfig        `config:"shared_credentials"`
	ConnectionString  *connectionStringConfig `config:"connection_string"`
	OAuth2            *OAuth2Config           `config:"oauth2"`
}

type connectionStringConfig struct {
	URI string `config:"uri"`
}
type sharedKeyConfig struct {
	AccountKey string `config:"account_key"`
}

type OAuth2Config struct {
	ClientID     string `config:"client_id"`
	ClientSecret string `config:"client_secret"`
	TenantID     string `config:"tenant_id"`
	// clientOptions is used internally for testing purposes only
	clientOptions azcore.ClientOptions
}

// isConfigEmpty checks if the provided configuration value is empty.
// It uses reflection to determine if the value is empty based on its kind.
// This function is generic and can handle any type T.
// Since this runs only once per source during initialization,
// it should not be a performance concern.
func isConfigEmpty[T any](value T) bool {
	v := reflect.ValueOf(value)
	return isEmpty(v)
}

// isEmpty checks if a reflect.Value is empty.
// It handles various types including pointers, slices, maps, structs, arrays, and basic types.
// It returns true if the value is empty, false otherwise.
func isEmpty(v reflect.Value) bool {
	// Handles cases like reflect.ValueOf(nil) where nil is untyped,
	// or an uninitialized interface variable.
	if !v.IsValid() {
		return true
	}

	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		// v.IsNil() checks if the pointer or interface is nil.
		// If it is nil, we consider it empty and return.
		if v.IsNil() {
			return true
		}
		return isEmpty(v.Elem())

	case reflect.Slice, reflect.Map:
		return v.IsNil() || v.Len() == 0

	case reflect.Struct:
		// Recursively check each field.
		for i := 0; i < v.NumField(); i++ {
			if !isEmpty(v.Field(i)) {
				return false
			}
		}
		return true

	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if !isEmpty(v.Index(i)) {
				return false
			}
		}
		return true

	// 'default:' handles basic types like int, string, bool, float, complex etc.
	default:
		return v.IsZero()
	}
}

func defaultConfig() config {
	return config{
		AccountName: "some_account",
	}
}

func (c config) Validate() error {
	if c.Auth.OAuth2 != nil && (c.Auth.OAuth2.ClientID == "" || c.Auth.OAuth2.ClientSecret == "" || c.Auth.OAuth2.TenantID == "") {
		return errors.New("client_id, client_secret and tenant_id are required for OAuth2 auth")
	}
	return nil
}
