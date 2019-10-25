// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transpiler

// MapVisitor visit the Tree and return a map[string]interface{}, this map can be serialized
// to a YAML document.
type MapVisitor struct {
	Content interface{}
}

// OnStr is called when we visit a StrVal.
func (m *MapVisitor) OnStr(v string) {
	m.Content = v
}

// OnInt is called when we visit a IntVal.
func (m *MapVisitor) OnInt(v int) {
	m.Content = v
}

// OnFloat is called when we visit a FloatVal.
func (m *MapVisitor) OnFloat(v float64) {
	m.Content = v
}

// OnBool is called when we visit a Bool.
func (m *MapVisitor) OnBool(v bool) {
	m.Content = v
}

// OnDict is called when we visit a Dict and return a VisitorDict.
func (m *MapVisitor) OnDict() VisitorDict {
	newMap := make(map[string]interface{})
	m.Content = newMap
	return &MapVisitorDict{Content: newMap}
}

// OnList is called when we visit a List and we return a VisitorList.
func (m *MapVisitor) OnList() VisitorList {
	m.Content = make([]interface{}, 0)
	return &MapVisitorList{MapVisitor: m}
}

// MapVisitorDict Visitor used for the visiting the Dict.
type MapVisitorDict struct {
	Content        map[string]interface{}
	lastVisitedKey string
}

// OnKey is called when we visit a key of a Dict.
func (m *MapVisitorDict) OnKey(s string) {
	m.lastVisitedKey = s
}

// OnValue is called when we visit a value of a Dict.
func (m *MapVisitorDict) OnValue(v Visitor) {
	visitor := v.(*MapVisitor)
	m.Content[m.lastVisitedKey] = visitor.Content
}

// Visitor returns a MapVisitor.
func (m *MapVisitorDict) Visitor() Visitor {
	return &MapVisitor{}
}

// OnComplete is called when you are done visiting the current Dict.
func (m *MapVisitorDict) OnComplete() {}

// MapVisitorList is a visitor to visit list.
type MapVisitorList struct {
	MapVisitor *MapVisitor
}

// OnComplete is called when we finish to visit a List.
func (m *MapVisitorList) OnComplete() {}

// OnValue is called when we visit a value and return a visitor.
func (m *MapVisitorList) OnValue(v Visitor) {
	visitor := v.(*MapVisitor)
	m.MapVisitor.Content = append(m.MapVisitor.Content.([]interface{}), visitor.Content)
}

// Visitor return a visitor.
func (m *MapVisitorList) Visitor() Visitor {
	return &MapVisitor{}
}
