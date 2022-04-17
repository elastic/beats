// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package o365audit

import (
	"fmt"
	"time"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
)

type apiError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (e apiError) getErrorStrings() (code, msg string) {
	const none = "(none)"
	code, msg = e.Error.Code, e.Error.Message
	if len(code) == 0 {
		code = none
	}
	if len(msg) == 0 {
		msg = none
	}
	return
}

func (e apiError) String() string {
	code, msg := e.getErrorStrings()
	return fmt.Sprintf("api error:%s %s", code, msg)
}

// ToBeatEvent returns a beat.Event representing the API error.
func (e apiError) ToBeatEvent() beat.Event {
	code, msg := e.getErrorStrings()
	return beat.Event{
		Timestamp: time.Now(),
		Fields: common.MapStr{
			"error": common.MapStr{
				"code":    code,
				"message": msg,
			},
			"event": common.MapStr{
				"kind": "pipeline_error",
			},
		},
	}
}

type content struct {
	Type       string    `json:"contentType"`
	ID         string    `json:"contentId"`
	URI        string    `json:"contentUri"`
	Created    time.Time `json:"contentCreated"`
	Expiration time.Time `json:"contentExpiration"`
}

type subscribeResponse struct {
	Status string `json:"status"`
}
