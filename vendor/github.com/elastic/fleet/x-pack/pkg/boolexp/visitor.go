// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package boolexp

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/antlr/antlr4/runtime/Go/antlr"

	"github.com/elastic/fleet/x-pack/pkg/boolexp/parser"
)

// Errors
var (
	ErrMissingVarStore = errors.New("no variable store defined")
)

type expVisitor struct {
	antlr.ParseTreeVisitor
	err        error
	methodsReg *MethodsReg
	vars       VarStore
}

func (v *expVisitor) Visit(tree antlr.ParseTree) interface{} {
	if v.hasErr() {
		return nil
	}

	switch value := tree.(type) {
	case *parser.ExpListContext:
		r := value.Accept(v)
		if v.hasErr() {
			return nil
		}
		return r.(bool)
	default:
		v.err = fmt.Errorf("unknown operation %T", tree.GetText())
		return false
	}
}

func (v *expVisitor) hasErr() bool {
	return v.err != nil
}

func (v *expVisitor) VisitExpList(ctx *parser.ExpListContext) interface{} {
	if v.hasErr() {
		return nil
	}
	r := ctx.Exp().Accept(v)
	return r.(bool)
}

func (v *expVisitor) VisitExpArithmeticNEQ(ctx *parser.ExpArithmeticNEQContext) interface{} {
	r, err := compareNEQ(ctx.GetLeft().Accept(v), ctx.GetRight().Accept(v))
	if err != nil {
		v.err = err
		return nil
	}
	return r
}

func (v *expVisitor) VisitExpArithmeticEQ(ctx *parser.ExpArithmeticEQContext) interface{} {
	r, err := compareEQ(ctx.GetLeft().Accept(v), ctx.GetRight().Accept(v))
	if err != nil {
		v.err = err
		return nil
	}
	return r
}

func (v *expVisitor) VisitExpArithmeticGTE(ctx *parser.ExpArithmeticGTEContext) interface{} {
	r, err := compareGTE(ctx.GetLeft().Accept(v), ctx.GetRight().Accept(v))
	if err != nil {
		v.err = err
	}
	return r
}

func (v *expVisitor) VisitExpArithmeticLTE(ctx *parser.ExpArithmeticLTEContext) interface{} {
	r, err := compareLTE(ctx.GetLeft().Accept(v), ctx.GetRight().Accept(v))
	if err != nil {
		v.err = err
	}
	return r
}

func (v *expVisitor) VisitExpArithmeticGT(ctx *parser.ExpArithmeticGTContext) interface{} {
	r, err := compareGT(ctx.GetLeft().Accept(v), ctx.GetRight().Accept(v))
	if err != nil {
		v.err = err
	}
	return r
}

func (v *expVisitor) VisitExpText(ctx *parser.ExpTextContext) interface{} {
	return str(ctx.GetText())
}

func (v *expVisitor) VisitExpNumber(ctx *parser.ExpNumberContext) interface{} {
	i, err := strconv.Atoi(ctx.GetText())
	if err != nil {
		v.err = fmt.Errorf("could not convert %s to an integer", ctx.GetText())
		return nil
	}
	return i
}

func (v *expVisitor) VisitExpFloat(ctx *parser.ExpFloatContext) interface{} {
	i, err := strconv.ParseFloat(ctx.GetText(), 64)
	if err != nil {
		v.err = fmt.Errorf("could not convert %s to a float", ctx.GetText())
		return nil
	}
	return i
}

func (v *expVisitor) VisitExpLogicalAnd(ctx *parser.ExpLogicalAndContext) interface{} {
	r, err := logicalAND(ctx.GetLeft().Accept(v), ctx.GetRight().Accept(v))
	if err != nil {
		v.err = err
		return nil
	}
	return r
}

func (v *expVisitor) VisitExpLogicalOR(ctx *parser.ExpLogicalORContext) interface{} {
	r, err := logicalOR(ctx.GetLeft().Accept(v), ctx.GetRight().Accept(v))
	if err != nil {
		v.err = err
		return nil
	}
	return r
}

func (v *expVisitor) VisitExpInParen(ctx *parser.ExpInParenContext) interface{} {
	return ctx.Exp().Accept(v)
}

func (v *expVisitor) VisitExpBoolean(ctx *parser.ExpBooleanContext) interface{} {
	b, err := strconv.ParseBool(ctx.GetText())
	if err != nil {
		v.err = fmt.Errorf("could not convert the value %s to a boolean", ctx.GetText())
		return nil
	}
	return b
}

func (v *expVisitor) VisitExpFunction(ctx *parser.ExpFunctionContext) interface{} {
	name := ctx.METHODNAME().GetText()
	method, ok := v.methodsReg.Lookup(name)
	if !ok {
		v.err = fmt.Errorf("call to unknown function %s", name)
		return nil
	}

	var err error
	var val interface{}
	if ctx.Arguments() != nil {
		args := ctx.Arguments().Accept(v).([]interface{})
		val, err = method.Func(args)
	} else {
		val, err = method.Func(make([]interface{}, 0))
	}

	if err != nil {
		v.err = err
		return nil
	}
	return val
}

func (v *expVisitor) VisitExpArithmeticLT(ctx *parser.ExpArithmeticLTContext) interface{} {
	r, err := compareLT(ctx.GetLeft().Accept(v), ctx.GetRight().Accept(v))
	if err != nil {
		v.err = err
	}
	return r
}

func (v *expVisitor) VisitBoolean(ctx *parser.BooleanContext) interface{} {
	return true
}

func (v *expVisitor) VisitArguments(ctx *parser.ArgumentsContext) interface{} {
	var args []interface{}

	for _, val := range ctx.AllExp() {
		args = append(args, val.Accept(v))
	}

	return args
}

func (v *expVisitor) VisitExpNot(ctx *parser.ExpNotContext) interface{} {
	r := ctx.Exp().Accept(v)
	if v.hasErr() {
		return nil
	}

	val, ok := r.(bool)
	if !ok {
		v.err = errors.New("value is not a boolean")
		return nil
	}

	return !val
}

func (v *expVisitor) VisitExpVariable(ctx *parser.ExpVariableContext) interface{} {
	if v.vars == nil {
		v.err = ErrMissingVarStore
		return nil
	}

	variable := ctx.VARIABLE().GetText()
	variable = variable[3 : len(variable)-2]

	val, ok := v.vars.Lookup(variable)
	if !ok {
		v.err = fmt.Errorf("unknown variable %s", variable)
		return nil
	}
	return val
}

func str(s string) string {
	if len(s) <= 2 {
		return ""
	}
	return s[1 : len(s)-1]
}
