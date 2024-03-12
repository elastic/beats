// Tencent is pleased to support the open source community by making bkunifylogbeat 蓝鲸日志采集器 available.
//
// Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
//
// bkunifylogbeat 蓝鲸日志采集器 is licensed under the MIT License.
//
// License for bkunifylogbeat 蓝鲸日志采集器:
// --------------------------------------------------------------------
// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated
// documentation files (the "Software"), to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software,
// and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all copies or substantial
// portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT
// LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN
// NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
// SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package syslog

import (
	"fmt"
	"regexp"
	"strings"
)

type MatchFunc func(text string) bool

func equal(value string) (MatchFunc, error) {
	matcher := func(text string) bool {
		return text == value
	}
	return matcher, nil
}

func notEqual(value string) (MatchFunc, error) {
	matcher := func(text string) bool {
		return text != value
	}
	return matcher, nil
}

func include(value string) (MatchFunc, error) {
	matcher := func(text string) bool {
		return strings.Contains(text, value)
	}
	return matcher, nil
}

func exclude(value string) (MatchFunc, error) {
	matcher := func(text string) bool {
		return !strings.Contains(text, value)
	}
	return matcher, nil
}

func regex(value string) (MatchFunc, error) {
	pattern, err := regexp.Compile(value)
	if err != nil {
		return nil, err
	}
	matcher := func(text string) bool {
		return pattern.MatchString(text)
	}
	return matcher, nil
}

func nRegex(value string) (MatchFunc, error) {
	pattern, err := regexp.Compile(value)
	if err != nil {
		return nil, err
	}
	matcher := func(text string) bool {
		return !pattern.MatchString(text)
	}
	return matcher, nil
}

const (
	opEq       = "eq"
	opNeq      = "neq"
	opEqual    = "="
	opNotEqual = "!="
	opInclude  = "include"
	opExclude  = "exclude"
	opRegex    = "regex"
	opNregex   = "nregex"
)

func getOperationFunc(op string, value string) (MatchFunc, error) {
	switch op {
	case opEqual, opEq:
		return equal(value)
	case opNotEqual, opNeq:
		return notEqual(value)
	case opInclude:
		return include(value)
	case opExclude:
		return exclude(value)
	case opRegex:
		return regex(value)
	case opNregex:
		return nRegex(value)
	default:
		return nil, fmt.Errorf("op:[%s] is not supported", op)
	}
}
