// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"encoding/json"
	"errors"

	"github.com/elastic/beats/v7/libbeat/logp"
)

type encoderFunc func(v interface{}) ([]byte, error)

type decoderFunc func(p []byte, dst interface{}) error

var (
	registeredEncoders = map[string]encoderFunc{}
	registeredDecoders = map[string]decoderFunc{}
	defaultEncoder     = json.Marshal
	defaultDecoder     = json.Unmarshal
)

func registerEncoder(contentType string, enc encoderFunc) error {
	if contentType == "" {
		return errors.New("content-type can't be empty")
	}

	if enc == nil {
		return errors.New("encoder can't be nil")
	}

	if _, found := registeredEncoders[contentType]; found {
		return errors.New("already registered")
	}

	registeredEncoders[contentType] = enc

	return nil
}

func registerDecoder(contentType string, dec decoderFunc) error {
	if contentType == "" {
		return errors.New("content-type can't be empty")
	}

	if dec == nil {
		return errors.New("decoder can't be nil")
	}

	if _, found := registeredDecoders[contentType]; found {
		return errors.New("already registered")
	}

	registeredDecoders[contentType] = dec

	return nil
}

func encode(contentType string, v interface{}) ([]byte, error) {
	enc, found := registeredEncoders[contentType]
	if !found {
		return defaultEncoder(v)
	}
	return enc(v)
}

func decode(contentType string, p []byte, v interface{}) error {
	dec, found := registeredDecoders[contentType]
	if !found {
		return defaultDecoder(p, v)
	}
	return dec(p, v)
}

func registerEncoders() {
	log := logp.L().Named(logName)
	log.Debug(registerEncoder("application/json", json.Marshal))
}

func registerDecoders() {
	log := logp.L().Named(logName)
	log.Debug(registerDecoder("application/json", json.Unmarshal))
}
