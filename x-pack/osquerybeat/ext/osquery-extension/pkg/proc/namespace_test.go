// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows
// +build !windows

package proc

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var ino = "26041992"

func TestParseNamespaceIno(t *testing.T) {
	testCases := []struct {
		nsLink string
		assert func(string, error)
	}{
		{
			fmt.Sprintf("pid:[%s]", ino),
			func(result string, err error) {
				assert.Nil(t, err)
				assert.Equal(t, result, "26041992")
			}},
		{
			fmt.Sprintf("pid:%s", ino),
			func(result string, err error) {
				assert.Error(t, err)
			}},
		{
			fmt.Sprintf("pid:[%s", ino),
			func(result string, err error) {
				assert.Error(t, err)
			}},
		{
			fmt.Sprintf("pid:%s]", ino),
			func(result string, err error) {
				assert.Error(t, err)
			}},
		{
			fmt.Sprintf("pid[%s]", ino),
			func(result string, err error) {
				assert.Error(t, err)
			}},
		{
			fmt.Sprintf("pid%s", ino),
			func(result string, err error) {
				assert.Error(t, err)
			}},
		{
			fmt.Sprintf("pid:[%s]:[%s]", ino, ino),
			func(result string, err error) {
				assert.Error(t, err)
			}},
		{
			"",
			func(result string, err error) {
				assert.Error(t, err)
			}},
		{
			"pid:",
			func(result string, err error) {
				assert.Error(t, err)
			}},
		{
			"pid:[]",
			func(result string, err error) {
				assert.Error(t, err)
			}},
		{
			"pid:[mock]",
			func(result string, err error) {
				assert.Error(t, err)
			}},
	}

	for _, testCase := range testCases {
		result, err := parseNamespaceIno(testCase.nsLink)
		testCase.assert(result, err)
	}
}
