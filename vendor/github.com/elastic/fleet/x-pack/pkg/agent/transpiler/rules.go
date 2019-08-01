// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transpiler

import (
	"fmt"
	"reflect"
	"regexp"
)

// RuleList is a container that allow the same tree to be executed on multiple defined Rule.
type RuleList struct {
	Rules []Rule
}

// Rule defines a rule that can be Applied on the Tree.
type Rule interface {
	Apply(*AST) error
}

// Apply applies a list of rules over the same tree and use the result of the previous execution
// as the input of the next rule, will return early if any error is raise during the execution.
func (rs *RuleList) Apply(ast *AST) error {
	var err error
	for _, rule := range rs.Rules {
		err = rule.Apply(ast)
		if err != nil {
			return err
		}
	}

	return nil
}

// RenameRule takes a selectors and will rename the last path of a Selector to a new name.
type RenameRule struct {
	path     Selector
	renameTo string
}

// Apply renames the last items of a Selector to a new name and keep all the other values and will
// return an error on failure.
func (r *RenameRule) Apply(ast *AST) error {
	// Skip rename when node is not found.
	node, ok := Lookup(ast, r.path)
	if !ok {
		return nil
	}

	n, ok := node.(*Key)
	if !ok {
		return fmt.Errorf("cannot rename, invalid type expected 'Key' received '%T'", node)
	}
	n.name = r.renameTo
	return nil
}

// Rename creates a rename rule.
func Rename(path Selector, renameTo string) *RenameRule {
	return &RenameRule{path: path, renameTo: renameTo}
}

// CopyRule take a from Selector and a destination selector and will insert an existing node into
// the destination, will return an errors if the types are incompatible.
type CopyRule struct {
	from Selector
	to   Selector
}

// Copy creates a copy rule.
func Copy(from, to Selector) *CopyRule {
	return &CopyRule{from: from, to: to}
}

// Apply copy a part of a tree into a new destination.
func (r CopyRule) Apply(ast *AST) error {
	node, ok := Lookup(ast, r.from)
	// skip when the `from` node is not found.
	if !ok {
		return nil
	}

	if err := Insert(ast, node, r.to); err != nil {
		return err
	}

	return nil
}

// TranslateKV is a place holder for mass replacement of values in the tree.
type TranslateKV struct {
	K interface{}
	V interface{}
}

// TranslateRule take a selector and will try to replace any values that match the translation
// table.
type TranslateRule struct {
	path   Selector
	mapper []TranslateKV
}

// Translate create a translation rule.
func Translate(path Selector, mapper []TranslateKV) *TranslateRule {
	return &TranslateRule{path: path, mapper: mapper}
}

// Apply translates matching elements of a translation table for a specific selector.
func (r *TranslateRule) Apply(ast *AST) error {
	// Skip translate when node is not found.
	node, ok := Lookup(ast, r.path)
	if !ok {
		return nil
	}

	n, ok := node.(*Key)
	if !ok {
		return fmt.Errorf("cannot rename, invalid type expected 'Key' received '%T'", node)
	}

	for _, kv := range r.mapper {
		if kv.K == n.Value().(Node).Value() {
			val := reflect.ValueOf(kv.V)
			nodeVal, err := load(val)
			if err != nil {
				return err
			}
			n.value = nodeVal
		}
	}

	return nil
}

// TranslateWithRegexpRule take a selector and will try to replace using the regular expression.
type TranslateWithRegexpRule struct {
	path Selector
	re   *regexp.Regexp
	with string
}

// TranslateWithRegexp create a translation rule.
func TranslateWithRegexp(path Selector, re *regexp.Regexp, with string) *TranslateWithRegexpRule {
	return &TranslateWithRegexpRule{path: path, re: re, with: with}
}

// Apply translates matching elements of a translation table for a specific selector.
func (r *TranslateWithRegexpRule) Apply(ast *AST) error {
	// Skip translate when node is not found.
	node, ok := Lookup(ast, r.path)
	if !ok {
		return nil
	}

	n, ok := node.(*Key)
	if !ok {
		return fmt.Errorf("cannot rename, invalid type expected 'Key' received '%T'", node)
	}

	candidate, ok := n.value.(Node).Value().(string)
	if !ok {
		return fmt.Errorf("cannot filter on value expected 'string' and received %T", candidate)
	}

	s := r.re.ReplaceAllString(candidate, r.with)
	val := reflect.ValueOf(s)
	nodeVal, err := load(val)
	if err != nil {
		return err
	}

	n.value = nodeVal

	return nil
}

// MapRule allow to apply mutliples rules on a subset of a Tree based on a provided selector.
type MapRule struct {
	path  Selector
	rules []Rule
}

// Map creates a new map rule.
func Map(path Selector, rules ...Rule) *MapRule {
	return &MapRule{path: path, rules: rules}
}

// Apply maps multiples rules over a subset of the tree.
func (r *MapRule) Apply(ast *AST) error {
	node, ok := Lookup(ast, r.path)
	// Skip map  when node is not found.
	if !ok {
		return nil
	}

	n, ok := node.(*Key)
	if !ok {
		return fmt.Errorf(
			"cannot iterate over node, invalid type expected 'Key' received '%T'",
			node,
		)
	}

	l, ok := n.Value().(*List)
	if !ok {
		return fmt.Errorf(
			"cannot iterate over node, invalid type expected 'List' received '%T'",
			node,
		)
	}

	values := l.Value().([]Node)

	for idx, item := range values {
		newAST := &AST{root: item}
		for _, rule := range r.rules {
			err := rule.Apply(newAST)
			if err != nil {
				return err
			}
			values[idx] = newAST.root
		}
	}
	return nil
}

// FilterRule allows to filter the tree and return only a subset of selectors.
type FilterRule struct {
	selectors []Selector
}

// Filter returns a new Filter Rule.
func Filter(selectors ...Selector) *FilterRule {
	return &FilterRule{selectors: selectors}
}

// Apply filters a Tree based on list of selectors.
func (r *FilterRule) Apply(ast *AST) error {
	mergedAST := &AST{root: &Dict{}}
	var err error
	for _, selector := range r.selectors {
		newAST, ok := Select(ast, selector)
		if !ok {
			continue
		}
		mergedAST, err = Combine(mergedAST, newAST)
		if err != nil {
			return err
		}
	}
	ast.root = mergedAST.root
	return nil
}

// FilterValuesRule allows to filter the tree and return only a subset of selectors with a predefined set of values.
type FilterValuesRule struct {
	selector Selector
	key      Selector
	values   []interface{}
}

// FilterValues returns a new FilterValues Rule.
func FilterValues(selector Selector, key Selector, values ...interface{}) *FilterValuesRule {
	return &FilterValuesRule{selector: selector, key: key, values: values}
}

// Apply filters a Tree based on list of selectors.
func (r *FilterValuesRule) Apply(ast *AST) error {
	node, ok := Lookup(ast, r.selector)
	// Skip map  when node is not found.
	if !ok {
		return nil
	}

	n, ok := node.(*Key)
	if !ok {
		return fmt.Errorf(
			"cannot iterate over node, invalid type expected 'Key' received '%T'",
			node,
		)
	}

	l, ok := n.Value().(*List)
	if !ok {
		return fmt.Errorf(
			"cannot iterate over node, invalid type expected 'List' received '%T'",
			node,
		)
	}

	values := l.Value().([]Node)
	var newNodes []Node

	for idx := 0; idx < len(values); idx++ {
		item := values[idx]
		newRoot := &AST{root: item}

		newAST, ok := Lookup(newRoot, r.key)
		if !ok {
			newNodes = append(newNodes, item)
			continue
		}

		// filter values
		n, ok := newAST.(*Key)
		if !ok {
			return fmt.Errorf("cannot filter on value, invalid type expected 'Key' received '%T'", newAST)
		}

		if n.name != r.key {
			newNodes = append(newNodes, item)
			continue
		}

		for _, v := range r.values {
			if v == n.value.(Node).Value() {
				newNodes = append(newNodes, item)
				break
			}
		}

	}

	l.value = newNodes
	n.value = l
	return nil
}

// FilterValuesWithRegexpRule allows to filter the tree and return only a subset of selectors with
// a regular expression.
type FilterValuesWithRegexpRule struct {
	selector Selector
	key      Selector
	re       *regexp.Regexp
}

// FilterValuesWithRegexp returns a new FilterValuesWithRegexp Rule.
func FilterValuesWithRegexp(
	selector Selector,
	key Selector,
	re *regexp.Regexp,
) *FilterValuesWithRegexpRule {
	return &FilterValuesWithRegexpRule{selector: selector, key: key, re: re}
}

// Apply filters a Tree based on list of selectors.
func (r *FilterValuesWithRegexpRule) Apply(ast *AST) error {
	node, ok := Lookup(ast, r.selector)
	// Skip map  when node is not found.
	if !ok {
		return nil
	}

	n, ok := node.(*Key)
	if !ok {
		return fmt.Errorf(
			"cannot iterate over node, invalid type expected 'Key' received '%T'",
			node,
		)
	}

	l, ok := n.Value().(*List)
	if !ok {
		return fmt.Errorf(
			"cannot iterate over node, invalid type expected 'List' received '%T'",
			node,
		)
	}

	values := l.Value().([]Node)
	var newNodes []Node

	for idx := 0; idx < len(values); idx++ {
		item := values[idx]
		newRoot := &AST{root: item}

		newAST, ok := Lookup(newRoot, r.key)
		if !ok {
			newNodes = append(newNodes, item)
			continue
		}

		// filter values
		n, ok := newAST.(*Key)
		if !ok {
			return fmt.Errorf("cannot filter on value, invalid type expected 'Key' received '%T'", newAST)
		}

		if n.name != r.key {
			newNodes = append(newNodes, item)
			continue
		}

		candidate, ok := n.value.(Node).Value().(string)
		if !ok {
			return fmt.Errorf("cannot filter on value expected 'string' and received %T", candidate)
		}

		if r.re.MatchString(candidate) {
			newNodes = append(newNodes, item)
		}
	}

	l.value = newNodes
	n.value = l
	return nil
}

// NewRuleList returns a new list of rules to be executed.
func NewRuleList(rules ...Rule) *RuleList {
	return &RuleList{Rules: rules}
}
