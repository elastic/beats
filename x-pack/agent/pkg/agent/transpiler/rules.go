// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transpiler

import (
	"fmt"
	"reflect"
	"regexp"

	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/errors"
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
func (r *RuleList) Apply(ast *AST) error {
	var err error
	for _, rule := range r.Rules {
		err = rule.Apply(ast)
		if err != nil {
			return err
		}
	}

	return nil
}

// MarshalYAML marsharl a rule list to YAML.
func (r *RuleList) MarshalYAML() (interface{}, error) {
	doc := make([]map[string]Rule, 0, len(r.Rules))

	for _, rule := range r.Rules {
		var name string
		switch rule.(type) {
		case *CopyRule:
			name = "copy"
		case *RenameRule:
			name = "rename"
		case *TranslateRule:
			name = "translate"
		case *TranslateWithRegexpRule:
			name = "translate_with_regexp"
		case *MapRule:
			name = "map"
		case *FilterRule:
			name = "filter"
		case *FilterValuesRule:
			name = "filter_values"
		case *FilterValuesWithRegexpRule:
			name = "filter_values_with_regexp"
		case *ExtractListItemRule:
			name = "extract_list_items"

		default:
			return nil, fmt.Errorf("unknown rule of type %T", rule)
		}

		subdoc := map[string]Rule{
			name: rule,
		}

		doc = append(doc, subdoc)
	}
	return doc, nil
}

// UnmarshalYAML unmashal a YAML document into a RuleList.
func (r *RuleList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var unpackTo []map[string]interface{}

	err := unmarshal(&unpackTo)
	if err != nil {
		return err
	}

	// NOTE(ph): this is a bit of a hack because I want to make sure
	// the unpack strategy stay in the struct implementation and yaml
	// doesn't have a RawMessage similar to the JSON package, so partial unpack
	// is not possible.
	unpack := func(in interface{}, out interface{}) error {
		b, err := yaml.Marshal(in)
		if err != nil {
			return err
		}
		return yaml.Unmarshal(b, out)
	}

	var rules []Rule

	for _, m := range unpackTo {
		ks := keys(m)
		if len(ks) > 1 {
			return fmt.Errorf("unknown rule identifier, expecting one identifier and received %d", len(ks))
		}

		name := ks[0]
		fields := m[name]

		var r Rule
		switch name {
		case "copy":
			r = &CopyRule{}
			if err := unpack(fields, r); err != nil {
				return err
			}
		case "rename":
			r = &RenameRule{}
			if err := unpack(fields, r); err != nil {
				return err
			}
		case "translate":
			r = &TranslateRule{}
			if err := unpack(fields, r); err != nil {
				return err
			}
		case "translate_with_regexp":
			r = &TranslateWithRegexpRule{}
			if err := unpack(fields, r); err != nil {
				return err
			}
		case "map":
			r = &MapRule{}
			if err := unpack(fields, r); err != nil {
				return err
			}
		case "filter":
			r = &FilterRule{}
			if err := unpack(fields, r); err != nil {
				return err
			}
		case "filter_values":
			r = &FilterValuesRule{}
			if err := unpack(fields, r); err != nil {
				return err
			}
		case "filter_values_with_regexp":
			r = &FilterValuesWithRegexpRule{}
			if err := unpack(fields, r); err != nil {
				return err
			}
		case "extract_list_items":
			r = &ExtractListItemRule{}
			if err := unpack(fields, r); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unkown rule of type %s", name)
		}

		rules = append(rules, r)
	}
	r.Rules = rules
	return nil
}

// ExtractListItemRule extract items with specified name from a list of maps.
// The result is store in a new array.
// Example:
// Source: {items: []List{ map{"key": "val1"}, map{"key", "val2"} } }
// extract-list-item -path:items -item:key -to:keys
// result:
// {items: []List{ map{"key": "val1"}, map{"key", "val2"} }, keys: []List {"val1", "val2"} }
type ExtractListItemRule struct {
	Path Selector
	Item string
	To   string
}

// Apply extracts items from array.
func (r *ExtractListItemRule) Apply(ast *AST) error {
	node, found := Lookup(ast, r.Path)
	if !found {
		return nil
	}

	nodeVal := node.Value()
	if nodeVal == nil {
		return nil
	}

	l, isList := nodeVal.(*List)
	if !isList {
		return nil
	}

	newList := &List{
		value: make([]Node, 0, len(l.value)),
	}

	for _, n := range l.value {
		in, found := n.Find(r.Item)
		if !found {
			continue
		}

		vn, ok := in.Value().(Node)
		if !ok {
			continue
		}

		newList.value = append(newList.value, vn.Clone())
	}

	return Insert(ast, newList, r.To)
}

func ExtractListItem(path Selector, item, target string) *ExtractListItemRule {
	return &ExtractListItemRule{
		Path: path,
		Item: item,
		To:   target,
	}
}

// RenameRule takes a selectors and will rename the last path of a Selector to a new name.
type RenameRule struct {
	From Selector
	To   string
}

// Apply renames the last items of a Selector to a new name and keep all the other values and will
// return an error on failure.
func (r *RenameRule) Apply(ast *AST) error {
	// Skip rename when node is not found.
	node, ok := Lookup(ast, r.From)
	if !ok {
		return nil
	}

	n, ok := node.(*Key)
	if !ok {
		return fmt.Errorf("cannot rename, invalid type expected 'Key' received '%T'", node)
	}
	n.name = r.To
	return nil
}

// Rename creates a rename rule.
func Rename(from Selector, to string) *RenameRule {
	return &RenameRule{From: from, To: to}
}

// CopyRule take a from Selector and a destination selector and will insert an existing node into
// the destination, will return an errors if the types are incompatible.
type CopyRule struct {
	From Selector
	To   Selector
}

// Copy creates a copy rule.
func Copy(from, to Selector) *CopyRule {
	return &CopyRule{From: from, To: to}
}

// Apply copy a part of a tree into a new destination.
func (r CopyRule) Apply(ast *AST) error {
	node, ok := Lookup(ast, r.From)
	// skip when the `from` node is not found.
	if !ok {
		return nil
	}

	if err := Insert(ast, node, r.To); err != nil {
		return err
	}

	return nil
}

// TranslateRule take a selector and will try to replace any values that match the translation
// table.
type TranslateRule struct {
	Path   Selector
	Mapper map[string]interface{}
}

// Translate create a translation rule.
func Translate(path Selector, mapper map[string]interface{}) *TranslateRule {
	return &TranslateRule{Path: path, Mapper: mapper}
}

// Apply translates matching elements of a translation table for a specific selector.
func (r *TranslateRule) Apply(ast *AST) error {
	// Skip translate when node is not found.
	node, ok := Lookup(ast, r.Path)
	if !ok {
		return nil
	}

	n, ok := node.(*Key)
	if !ok {
		return fmt.Errorf("cannot rename, invalid type expected 'Key' received '%T'", node)
	}

	for k, v := range r.Mapper {
		if k == n.Value().(Node).Value() {
			val := reflect.ValueOf(v)
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
	Path Selector
	Re   *regexp.Regexp
	With string
}

// MarshalYAML marshal a TranslateWithRegexpRule into a YAML document.
func (r *TranslateWithRegexpRule) MarshalYAML() (interface{}, error) {
	return map[string]interface{}{
		"path": r.Path,
		"re":   r.Re.String(),
		"with": r.With,
	}, nil
}

// UnmarshalYAML unmarshal a YAML document into a TranslateWithRegexpRule.
func (r *TranslateWithRegexpRule) UnmarshalYAML(unmarshal func(interface{}) error) error {
	tmp := struct {
		Path string
		Re   string
		With string
	}{}

	if err := unmarshal(&tmp); err != nil {
		return errors.New(err, "cannot unmarshal into a TranslateWithRegexpRule")
	}

	re, err := regexp.Compile(tmp.Re)
	if err != nil {
		errors.New(err, "invalid regular expression for TranslateWithRegexpRule")
	}

	*r = TranslateWithRegexpRule{
		Path: tmp.Path,
		Re:   re,
		With: tmp.With,
	}
	return nil
}

// TranslateWithRegexp create a translation rule.
func TranslateWithRegexp(path Selector, re *regexp.Regexp, with string) *TranslateWithRegexpRule {
	return &TranslateWithRegexpRule{Path: path, Re: re, With: with}
}

// Apply translates matching elements of a translation table for a specific selector.
func (r *TranslateWithRegexpRule) Apply(ast *AST) error {
	// Skip translate when node is not found.
	node, ok := Lookup(ast, r.Path)
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

	s := r.Re.ReplaceAllString(candidate, r.With)
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
	Path  Selector
	Rules []Rule
}

// Map creates a new map rule.
func Map(path Selector, rules ...Rule) *MapRule {
	return &MapRule{Path: path, Rules: rules}
}

// Apply maps multiples rules over a subset of the tree.
func (r *MapRule) Apply(ast *AST) error {
	node, ok := Lookup(ast, r.Path)
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
		for _, rule := range r.Rules {
			err := rule.Apply(newAST)
			if err != nil {
				return err
			}
			values[idx] = newAST.root
		}
	}
	return nil
}

// MarshalYAML marshal a MapRule into a YAML document.
func (r *MapRule) MarshalYAML() (interface{}, error) {
	rules, err := NewRuleList(r.Rules...).MarshalYAML()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"path":  r.Path,
		"rules": rules,
	}, nil
}

// UnmarshalYAML unmarshal a YAML document into a MapRule.
func (r *MapRule) UnmarshalYAML(unmarshal func(interface{}) error) error {
	tmp := struct {
		Path  string
		Rules RuleList
	}{}

	if err := unmarshal(&tmp); err != nil {
		return errors.New(err, "cannot unmarshal into a MapRule")
	}

	*r = MapRule{
		Path:  tmp.Path,
		Rules: tmp.Rules.Rules,
	}
	return nil
}

// FilterRule allows to filter the tree and return only a subset of selectors.
type FilterRule struct {
	Selectors []Selector
}

// Filter returns a new Filter Rule.
func Filter(selectors ...Selector) *FilterRule {
	return &FilterRule{Selectors: selectors}
}

// Apply filters a Tree based on list of selectors.
func (r *FilterRule) Apply(ast *AST) error {
	mergedAST := &AST{root: &Dict{}}
	var err error
	for _, selector := range r.Selectors {
		newAST, ok := Select(ast.Clone(), selector)
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
	Selector Selector
	Key      Selector
	Values   []interface{}
}

// FilterValues returns a new FilterValues Rule.
func FilterValues(selector Selector, key Selector, values ...interface{}) *FilterValuesRule {
	return &FilterValuesRule{Selector: selector, Key: key, Values: values}
}

// Apply filters a Tree based on list of selectors.
func (r *FilterValuesRule) Apply(ast *AST) error {
	node, ok := Lookup(ast, r.Selector)
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

		newAST, ok := Lookup(newRoot, r.Key)
		if !ok {
			newNodes = append(newNodes, item)
			continue
		}

		// filter values
		n, ok := newAST.(*Key)
		if !ok {
			return fmt.Errorf("cannot filter on value, invalid type expected 'Key' received '%T'", newAST)
		}

		if n.name != r.Key {
			newNodes = append(newNodes, item)
			continue
		}

		for _, v := range r.Values {
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
	Selector Selector
	Key      Selector
	Re       *regexp.Regexp
}

// FilterValuesWithRegexp returns a new FilterValuesWithRegexp Rule.
func FilterValuesWithRegexp(
	selector Selector,
	key Selector,
	re *regexp.Regexp,
) *FilterValuesWithRegexpRule {
	return &FilterValuesWithRegexpRule{Selector: selector, Key: key, Re: re}
}

// MarshalYAML marshal a FilterValuesWithRegexpRule into a YAML document.
func (r *FilterValuesWithRegexpRule) MarshalYAML() (interface{}, error) {
	return map[string]interface{}{
		"selector": r.Selector,
		"key":      r.Key,
		"re":       r.Re.String(),
	}, nil
}

// UnmarshalYAML unmarshal a YAML document into a FilterValuesWithRegexpRule.
func (r *FilterValuesWithRegexpRule) UnmarshalYAML(unmarshal func(interface{}) error) error {
	tmp := struct {
		Selector string
		Key      string
		Re       string
	}{}

	if err := unmarshal(&tmp); err != nil {
		return errors.New(err, "cannot unmarshal into a FilterValuesWithRegexpRule")
	}

	re, err := regexp.Compile(tmp.Re)
	if err != nil {
		errors.New(err, "invalid regular expression for FilterValuesWithRegexpRule")
	}
	*r = FilterValuesWithRegexpRule{
		Selector: tmp.Selector,
		Key:      tmp.Key,
		Re:       re,
	}

	return nil
}

// Apply filters a Tree based on list of selectors.
func (r *FilterValuesWithRegexpRule) Apply(ast *AST) error {
	node, ok := Lookup(ast, r.Selector)
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

		newAST, ok := Lookup(newRoot, r.Key)
		if !ok {
			newNodes = append(newNodes, item)
			continue
		}

		// filter values
		n, ok := newAST.(*Key)
		if !ok {
			return fmt.Errorf("cannot filter on value, invalid type expected 'Key' received '%T'", newAST)
		}

		if n.name != r.Key {
			newNodes = append(newNodes, item)
			continue
		}

		candidate, ok := n.value.(Node).Value().(string)
		if !ok {
			return fmt.Errorf("cannot filter on value expected 'string' and received %T", candidate)
		}

		if r.Re.MatchString(candidate) {
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

func keys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
