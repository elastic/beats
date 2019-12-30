// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package info

import (
	"bytes"
	"fmt"
	"io"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/storage"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const agentInfoKey = "agent-info"

type persistentAgentInfo struct {
	ID string `json:"ID" yaml:"ID"`
}

type ioStore interface {
	Save(io.Reader) error
	Load() (io.ReadCloser, error)
}

func generateAgentID() (string, error) {
	s := storage.NewEncryptedDiskStore(agentInfoKey, []byte(""))

	id := loadAgentID(s)
	if id != "" {
		return id, nil
	}

	uid, err := uuid.NewV4()
	if err != nil {
		return "", fmt.Errorf("error while generating UUID for agent: %v", err)
	}

	id = uid.String()

	if err := storeAgentID(s, id); err != nil {
		return "", errors.Wrap(err, "storing generated agent id")
	}

	return id, nil
}

func loadAgentID(s ioStore) string {
	r, err := s.Load()
	if err != nil {
		return ""
	}
	d := yaml.NewDecoder(r)

	id := &persistentAgentInfo{}
	if err := d.Decode(&id); err != nil {
		return ""
	}

	return id.ID
}

func storeAgentID(s ioStore, id string) error {
	ids := &persistentAgentInfo{
		ID: id,
	}

	r, err := yamlToReader(ids)
	if err != nil {
		return err
	}

	return s.Save(r)
}

func yamlToReader(in interface{}) (io.Reader, error) {
	data, err := yaml.Marshal(in)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal to YAML")
	}
	return bytes.NewReader(data), nil
}
