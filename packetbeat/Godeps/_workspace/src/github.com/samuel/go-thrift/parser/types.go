// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package parser

type Type struct {
	Name      string
	KeyType   *Type // If map
	ValueType *Type // If map or list
}

type EnumValue struct {
	Name  string
	Value int
}

type Enum struct {
	Name   string
	Values map[string]*EnumValue
}

type Constant struct {
	Name  string
	Type  *Type
	Value interface{}
}

type Field struct {
	Id       int
	Name     string
	Optional bool
	Type     *Type
	Default  interface{}
}

type Struct struct {
	Name   string
	Fields []*Field
}

type Method struct {
	Comment    string
	Name       string
	Oneway     bool
	ReturnType *Type
	Arguments  []*Field
	Exceptions []*Field
}

type Service struct {
	Name    string
	Extends string
	Methods map[string]*Method
}

type Thrift struct {
	Includes   map[string]string // name -> unique identifier (absolute path generally)
	Typedefs   map[string]*Type
	Namespaces map[string]string
	Constants  map[string]*Constant
	Enums      map[string]*Enum
	Structs    map[string]*Struct
	Exceptions map[string]*Struct
	Services   map[string]*Service
}
