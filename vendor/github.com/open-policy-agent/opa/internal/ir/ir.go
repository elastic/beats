// Copyright 2018 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

// Package ir defines an intermediate representation (IR) for Rego.
//
// The IR specifies an imperative execution model for Rego policies similar to a
// query plan in traditional databases.
package ir

import (
	"fmt"

	"github.com/open-policy-agent/opa/types"
)

type (
	// Policy represents a planned policy query.
	Policy struct {
		Static *Static
		Plans  *Plans
		Funcs  *Funcs
	}

	// Static represents a static data segment that is indexed into by the policy.
	Static struct {
		Strings      []*StringConst
		BuiltinFuncs []*BuiltinFunc
		Files        []*StringConst
	}

	// BuiltinFunc represents a built-in function that may be required by the
	// policy.
	BuiltinFunc struct {
		Name string
		Decl *types.Function
	}

	// Plans represents a collection of named query plans to expose in the policy.
	Plans struct {
		Plans []*Plan
	}

	// Funcs represents a collection of planned functions to include in the
	// policy.
	Funcs struct {
		Funcs []*Func
	}

	// Func represents a named plan (function) that can be invoked. Functions
	// accept one or more parameters and return a value. By convention, the
	// input document and data documents are always passed as the first and
	// second arguments (respectively).
	Func struct {
		Name   string
		Params []Local
		Return Local
		Blocks []*Block // TODO(tsandall): should this be a plan?
		Path   []string // optional: if non-nil, include in data function tree
	}

	// Plan represents an ordered series of blocks to execute. Plan execution
	// stops when a return statement is reached. Blocks are executed in-order.
	Plan struct {
		Name   string
		Blocks []*Block
	}

	// Block represents an ordered sequence of statements to execute. Blocks are
	// executed until a return statement is encountered, a statement is undefined,
	// or there are no more statements. If all statements are defined but no return
	// statement is encountered, the block is undefined.
	Block struct {
		Stmts []Stmt
	}

	// Stmt represents an operation (e.g., comparison, loop, dot, etc.) to execute.
	Stmt interface {
		locationStmt
	}

	locationStmt interface {
		SetLocation(index, row, col int, file, text string)
		GetLocation() *Location
	}

	// Local represents a plan-scoped variable.
	//
	// TODO(tsandall): should this be int32 for safety?
	Local int

	// Const represents a constant value from the policy.
	Const interface {
		typeMarker()
	}

	// NullConst represents a null value.
	NullConst struct{}

	// BooleanConst represents a boolean value.
	BooleanConst struct {
		Value bool
	}

	// StringConst represents a string value.
	StringConst struct {
		Value string
	}

	// IntConst represents an integer constant.
	IntConst struct {
		Value int64
	}

	// FloatConst represents a floating-point constant.
	FloatConst struct {
		Value float64
	}
)

const (
	// Input is the local variable that refers to the global input document.
	Input Local = iota

	// Data is the local variable that refers to the global data document.
	Data

	// Unused is the free local variable that can be allocated in a plan.
	Unused
)

func (a *Policy) String() string {
	return "Policy"
}

func (a *Static) String() string {
	return fmt.Sprintf("Static (%d strings, %d files)", len(a.Strings), len(a.Files))
}

func (a *Funcs) String() string {
	return fmt.Sprintf("Funcs (%d funcs)", len(a.Funcs))
}

func (a *Func) String() string {
	return fmt.Sprintf("%v (%d params: %v, %d blocks, path: %v)", a.Name, len(a.Params), a.Params, len(a.Blocks), a.Path)
}

func (a *Plan) String() string {
	return fmt.Sprintf("Plan %v (%d blocks)", a.Name, len(a.Blocks))
}

func (a *Block) String() string {
	return fmt.Sprintf("Block (%d statements)", len(a.Stmts))
}

func (*BooleanConst) typeMarker() {}
func (*NullConst) typeMarker()    {}
func (*IntConst) typeMarker()     {}
func (*FloatConst) typeMarker()   {}
func (*StringConst) typeMarker()  {}

// ReturnLocalStmt represents a return statement that yields a local value.
type ReturnLocalStmt struct {
	Source Local

	Location
}

// CallStmt represents a named function call. The result should be stored in the
// result local.
type CallStmt struct {
	Func   string
	Args   []LocalOrConst
	Result Local

	Location
}

// CallDynamicStmt represents an indirect (data) function call. The result should
// be stored in the result local.
type CallDynamicStmt struct {
	Args   []Local
	Result Local
	Path   []LocalOrConst

	Location
}

// LocalOrConst is a tagged union of the two types, Local and StringIndex.
type LocalOrConst interface {
	fmt.Stringer
	localOrConst()
}

func (Local) localOrConst() {}
func (l Local) String() string {
	return fmt.Sprintf("Local<%d>", int(l))
}

// StringIndex represents the index into the plan's list of constant strings
// of a constant string.
type StringIndex int

func (StringIndex) localOrConst() {}
func (s StringIndex) String() string {
	return fmt.Sprintf("String<%d>", int(s))
}

// Bool represents a constant boolean.
type Bool bool

func (Bool) localOrConst() {}
func (b Bool) String() string {
	return fmt.Sprintf("Bool<%v>", bool(b))
}

// BlockStmt represents a nested block. Nested blocks and break statements can
// be used to short-circuit execution.
type BlockStmt struct {
	Blocks []*Block

	Location
}

func (a *BlockStmt) String() string {
	return fmt.Sprintf("BlockStmt (%d blocks) %v", len(a.Blocks), a.GetLocation())
}

// BreakStmt represents a jump out of the current block. The index specifies how
// many blocks to jump starting from zero (the current block). Execution will
// continue from the end of the block that is jumped to.
type BreakStmt struct {
	Index uint32

	Location
}

// DotStmt represents a lookup operation on a value (e.g., array, object, etc.)
// The source of a DotStmt may be a scalar value in which case the statement
// will be undefined.
type DotStmt struct {
	Source LocalOrConst
	Key    LocalOrConst
	Target Local

	Location
}

// LenStmt represents a length() operation on a local variable. The
// result is stored in the target local variable.
type LenStmt struct {
	Source LocalOrConst
	Target Local

	Location
}

// ScanStmt represents a linear scan over a composite value. The
// source may be a scalar in which case the block will never execute.
type ScanStmt struct {
	Source Local
	Key    Local
	Value  Local
	Block  *Block

	Location
}

// NotStmt represents a negated statement.
type NotStmt struct {
	Block *Block

	Location
}

// AssignIntStmt represents an assignment of an integer value to a
// local variable.
type AssignIntStmt struct {
	Value  int64
	Target Local

	Location
}

// AssignVarStmt represents an assignment of one local variable to another.
type AssignVarStmt struct {
	Source LocalOrConst
	Target Local

	Location
}

// AssignVarOnceStmt represents an assignment of one local variable to another.
// If the target is defined, execution aborts with a conflict error.
//
// TODO(tsandall): is there a better name for this?
type AssignVarOnceStmt struct {
	Target Local
	Source LocalOrConst

	Location
}

// ResetLocalStmt resets a local variable to 0.
type ResetLocalStmt struct {
	Target Local

	Location
}

// MakeNullStmt constructs a local variable that refers to a null value.
type MakeNullStmt struct {
	Target Local

	Location
}

// MakeNumberIntStmt constructs a local variable that refers to an integer value.
type MakeNumberIntStmt struct {
	Value  int64
	Target Local

	Location
}

// MakeNumberRefStmt constructs a local variable that refers to a number stored as a string.
type MakeNumberRefStmt struct {
	Index  int
	Target Local

	Location
}

// MakeArrayStmt constructs a local variable that refers to an array value.
type MakeArrayStmt struct {
	Capacity int32
	Target   Local

	Location
}

// MakeObjectStmt constructs a local variable that refers to an object value.
type MakeObjectStmt struct {
	Target Local

	Location
}

// MakeSetStmt constructs a local variable that refers to a set value.
type MakeSetStmt struct {
	Target Local

	Location
}

// EqualStmt represents an value-equality check of two local variables.
type EqualStmt struct {
	A LocalOrConst
	B LocalOrConst

	Location
}

// NotEqualStmt represents a != check of two local variables.
type NotEqualStmt struct {
	A LocalOrConst
	B LocalOrConst

	Location
}

// IsArrayStmt represents a dynamic type check on a local variable.
type IsArrayStmt struct {
	Source LocalOrConst

	Location
}

// IsObjectStmt represents a dynamic type check on a local variable.
type IsObjectStmt struct {
	Source LocalOrConst

	Location
}

// IsDefinedStmt represents a check of whether a local variable is defined.
type IsDefinedStmt struct {
	Source Local

	Location
}

// IsUndefinedStmt represents a check of whether local variable is undefined.
type IsUndefinedStmt struct {
	Source Local

	Location
}

// ArrayAppendStmt represents a dynamic append operation of a value
// onto an array.
type ArrayAppendStmt struct {
	Value LocalOrConst
	Array Local

	Location
}

// ObjectInsertStmt represents a dynamic insert operation of a
// key/value pair into an object.
type ObjectInsertStmt struct {
	Key    LocalOrConst
	Value  LocalOrConst
	Object Local

	Location
}

// ObjectInsertOnceStmt represents a dynamic insert operation of a key/value
// pair into an object. If the key already exists and the value differs,
// execution aborts with a conflict error.
type ObjectInsertOnceStmt struct {
	Key    LocalOrConst
	Value  LocalOrConst
	Object Local

	Location
}

// ObjectMergeStmt performs a recursive merge of two object values. If either of
// the locals refer to non-object values this operation will abort with a
// conflict error. Overlapping object keys are merged recursively.
type ObjectMergeStmt struct {
	A      Local
	B      Local
	Target Local

	Location
}

// SetAddStmt represents a dynamic add operation of an element into a set.
type SetAddStmt struct {
	Value LocalOrConst
	Set   Local

	Location
}

// WithStmt replaces the Local or a portion of the document referred to by the
// Local with the Value and executes the contained block. If the Path is
// non-empty, the Value is upserted into the Local. If the intermediate nodes in
// the Local referred to by the Path do not exist, they will be created. When
// the WithStmt finishes the Local is reset to it's original value.
type WithStmt struct {
	Local Local
	Path  []int
	Value LocalOrConst
	Block *Block

	Location
}

// NopStmt adds a nop instruction. Useful during development and debugging only.
type NopStmt struct {
	Location
}

// ResultSetAdd adds a value into the result set returned by the query plan.
type ResultSetAdd struct {
	Value Local

	Location
}

// Location records the filen index, and the row and column inside that file
// that a statement can be connected to.
type Location struct {
	Index      int // filename string constant index
	Col, Row   int
	file, text string // only used for debugging
}

// SetLocation sets the Location for a given Stmt.
func (l *Location) SetLocation(index, row, col int, file, text string) {
	*l = Location{
		Index: index,
		Row:   row,
		Col:   col,
		file:  file,
		text:  text,
	}
}

// GetLocation returns a Stmt's Location.
func (l *Location) GetLocation() *Location {
	return l
}
