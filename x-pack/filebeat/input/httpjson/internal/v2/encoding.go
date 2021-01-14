// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"encoding/json"
	"errors"

	"github.com/elastic/beats/v7/libbeat/logp"
)

type encoderFunc func(trReq transformable) ([]byte, error)

type decoderFunc func(p []byte, dst *response) error

var (
	registeredEncoders             = map[string]encoderFunc{}
	registeredDecoders             = map[string]decoderFunc{}
	defaultEncoder     encoderFunc = encodeAsJSON
	defaultDecoder     decoderFunc = decodeAsJSON
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

func encode(contentType string, trReq transformable) ([]byte, error) {
	enc, found := registeredEncoders[contentType]
	if !found {
		return defaultEncoder(trReq)
	}
	return enc(trReq)
}

func decode(contentType string, p []byte, dst *response) error {
	dec, found := registeredDecoders[contentType]
	if !found {
		return defaultDecoder(p, dst)
	}
	return dec(p, dst)
}

func registerEncoders() {
	log := logp.L().Named(logName)
	log.Debug(registerEncoder("application/json", encodeAsJSON))
}

func registerDecoders() {
	log := logp.L().Named(logName)
	log.Debug(registerDecoder("application/json", decodeAsJSON))
}

func encodeAsJSON(trReq transformable) ([]byte, error) {
	return json.Marshal(trReq.body())
}

func decodeAsJSON(p []byte, dst *response) error {
	return json.Unmarshal(p, &dst.body)
}
