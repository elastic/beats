// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package utils

import (
	"net/url"
	"sort"
	"strings"

	"golang.org/x/exp/maps"
)

type Supplier[T any, R any] func(T) R

type Predicate[T any] func(T) bool

type CheckedPredicate[T any] func(T) (bool, error)

func ParseArrayOfStrings(input string) []string {
	withoutBrackets := strings.Trim(input, "[]")

	if withoutBrackets == "" {
		return []string{}
	}

	elements := strings.Split(withoutBrackets, ",")

	trimmedNotEmptyElements := make([]string, 0)
	for _, e := range elements {
		trimmed := strings.TrimSpace(e)
		if trimmed != "" {
			trimmedNotEmptyElements = append(trimmedNotEmptyElements, trimmed)
		}

	}

	return trimmedNotEmptyElements
}

func MatchesAnyPattern(value string, patterns []string) bool {
	if len(patterns) == 0 || len(value) == 0 {
		return false
	}
	for _, pattern := range patterns {
		if strings.HasSuffix(pattern, "*") {
			withoutStar := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(value, withoutStar) {
				return true
			}
		} else if strings.Compare(pattern, value) == 0 {
			return true
		}
	}
	return false
}

func AnyMatchesAnyPattern(values []string, patterns []string) bool {
	for _, val := range values {
		if MatchesAnyPattern(val, patterns) {
			return true
		}
	}
	return false
}

func AddInt64OrNull(total *int64, increment *int64) *int64 {
	if increment == nil {
		return total
	}

	newTotal := total

	if total == nil {
		var val int64 = 0
		newTotal = &val
	}

	*newTotal += *increment

	return newTotal
}

func FilterAndMap[T any, R any](items []R, supplier Supplier[*R, T], predicate Predicate[*R]) []T {
	var list []T
	for _, item := range items {
		if predicate(&item) {
			list = append(list, supplier(&item))
		}
	}
	return list
}

func PartitionByMaxValue[T any](limit int, items []T, valueExtractor func(T) int) map[int][]T {
	sortedValues := make(map[int][]T)
	for _, item := range items {
		itemKey := valueExtractor(item)
		sortedValues[itemKey] = append(sortedValues[itemKey], item)
	}
	allKeys := maps.Keys(sortedValues)
	sort.Ints(allKeys)
	var sortedItems = make(map[int][]T)
	currentCapacity := 0
	cursor := 0
	for _, key := range allKeys {
		for _, val := range sortedValues[key] {
			currentCapacity += key
			if currentCapacity >= limit {
				cursor++
				currentCapacity = 0
			}
			sortedItems[cursor] = append(sortedItems[cursor], val)
		}
	}
	return sortedItems
}

func GetStringArrayFromArrayOrSingleValue(field interface{}) []string {
	switch value := field.(type) {
	case string:
		return []string{value}
	case []string:
		return value
	case []interface{}:
		var data []string
		for _, str := range value {
			data = append(data, GetStringArrayFromArrayOrSingleValue(str)...)
		}
		return data
	default:
		return nil
	}
}

func UrlEscapeNames(names []string, stringToExclude string) []string {
	result := make([]string, 0, len(names)) // Preallocate slice with the same length as input

	for _, name := range names {
		trimmedName := strings.TrimSpace(name)
		if !strings.Contains(trimmedName, stringToExclude) {
			result = append(result, url.PathEscape(trimmedName))
		}
	}

	return result
}
