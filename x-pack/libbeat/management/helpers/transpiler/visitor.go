// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transpiler

// Visitor defines the interface to use when visiting all the nodes in the Tree.
type Visitor interface {
	OnDict() VisitorDict
	OnList() VisitorList
	OnStr(string)
	OnInt(int)
	OnUInt(uint64)
	OnFloat(float64)
	OnBool(bool)
}

// VisitorDict to use when visiting a Dict.
type VisitorDict interface {
	OnKey(string)
	Visitor() Visitor
	OnValue(Visitor)
	OnComplete()
}

// VisitorList to use when visiting a List.
type VisitorList interface {
	OnValue(Visitor)
	Visitor() Visitor
	OnComplete()
}
