// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package jumplists

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists/parsers/olecfb"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists/parsers/resources"
)

type AutomaticJumpList struct {
	appId resources.ApplicationId
	olecfb *olecfb.Olecfb
}

func (a *AutomaticJumpList) Path() string {
	return a.olecfb.Path
}

func (a *AutomaticJumpList) AppId() resources.ApplicationId {
	return a.appId
}

func (a *AutomaticJumpList) Type() JumpListType {
	return JumpListTypeAutomatic
}

func NewAutomaticJumpList(filePath string, log *logger.Logger) (*AutomaticJumpList, error) {
	olecfb, err := olecfb.NewOlecfb(filePath, log)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Olecfb: %w", err)
	}
	return &AutomaticJumpList{
		appId: resources.GetAppIdFromFileName(filePath, log),
		olecfb: olecfb,
	}, nil
}

func GetAutomaticJumpLists(log *logger.Logger) ([]*AutomaticJumpList, error) {
	files, err := FindJumplistFiles(JumpListTypeAutomatic, log)
	if err != nil {
		return nil, err
	}
	var jumpLists []*AutomaticJumpList
	for _, file := range files {
		jumpList, err := NewAutomaticJumpList(file, log)
		if err != nil {
			log.Errorf("failed to parse Automatic Jump List: %v", err)
			continue
		}
		jumpLists = append(jumpLists, jumpList)
	}
	return jumpLists, nil
}
