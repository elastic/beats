// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package packer

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/errors"
)

// PackMap represents multiples files packed, the key represent the path of the file and raw bytes of
// the files.
type PackMap map[string][]byte

// Pack takes a patterns of multiples files and will read all of them, compress and encoded the data,
// it will return the encoded string, the list of files encoded and any errors.
func Pack(patterns ...string) (string, []string, error) {
	var encodedFiles []string

	pack := make(PackMap)
	for _, p := range patterns {
		files, err := filepath.Glob(p)
		if err != nil {
			return "", []string{}, errors.New(err, fmt.Sprintf("error while reading pattern %s", p))
		}
		for _, f := range files {
			b, err := ioutil.ReadFile(f)
			if err != nil {
				return "", []string{}, errors.New(err, fmt.Sprintf("cannot read file %s", f))
			}

			_, ok := pack[f]
			if ok {
				return "", []string{}, errors.New(fmt.Sprintf("file %s already packed", f))
			}

			encodedFiles = append(encodedFiles, f)
			pack[f] = b
		}
	}

	if len(pack) == 0 {
		return "", []string{}, fmt.Errorf("no files found with provided patterns: %s", strings.Join(patterns, ", "))
	}

	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	enc := json.NewEncoder(w)
	if err := enc.Encode(pack); err != nil {
		return "", []string{}, errors.New(err, "could not encode files")
	}
	// flush any buffers.
	w.Close()

	return base64.StdEncoding.EncodeToString(buf.Bytes()), encodedFiles, nil
}

// Unpack takes a Pack and return an uncompressed map with the raw bytes array.
func Unpack(pack string) (PackMap, error) {
	d, err := base64.StdEncoding.DecodeString(pack)
	if err != nil {
		return nil, errors.New(err, "error while decoding")
	}

	b := bytes.NewReader(d)
	r, err := zlib.NewReader(b)
	if err != nil {
		return nil, errors.New(err, "error while uncompressing")
	}
	defer r.Close()

	var uncompressed PackMap
	dec := json.NewDecoder(r)
	if err := dec.Decode(&uncompressed); err != nil {
		return nil, errors.New(err, "could no read the pack data")
	}

	return uncompressed, nil
}

// MustUnpack unpack the packs and will panic on error.
func MustUnpack(pack string) PackMap {
	v, err := Unpack(pack)
	if err != nil {
		panic(err)
	}
	return v
}

// MustPackFile will pack all the files matching the patterns and will panic on any errors.
func MustPackFile(patterns ...string) (string, []string) {
	v, files, err := Pack(patterns...)
	if err != nil {
		panic(err)
	}
	return v, files
}
