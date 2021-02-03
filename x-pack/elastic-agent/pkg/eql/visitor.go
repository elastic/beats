// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package eql

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/antlr/antlr4/runtime/Go/antlr"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/eql/parser"
)

// Errors
var (
	ErrMissingVarStore = errors.New("no variable store defined")
)

type null struct{}

// Null is returned when the variable doesn't exist.
var Null = &null{}

type expVisitor struct {
	antlr.ParseTreeVisitor
	err  error
	vars VarStore
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
		return nil
	}
	return r
}

func (v *expVisitor) VisitExpArithmeticLTE(ctx *parser.ExpArithmeticLTEContext) interface{} {
	r, err := compareLTE(ctx.GetLeft().Accept(v), ctx.GetRight().Accept(v))
	if err != nil {
		v.err = err
		return nil
	}
	return r
}

func (v *expVisitor) VisitExpArithmeticGT(ctx *parser.ExpArithmeticGTContext) interface{} {
	r, err := compareGT(ctx.GetLeft().Accept(v), ctx.GetRight().Accept(v))
	if err != nil {
		v.err = err
		return nil
	}
	return r
}

func (v *expVisitor) VisitExpArithmeticAddSub(ctx *parser.ExpArithmeticAddSubContext) interface{} {
	var r interface{}
	var err error
	if ctx.ADD() != nil {
		r, err = mathAdd(ctx.GetLeft().Accept(v), ctx.GetRight().Accept(v))
	} else if ctx.SUB() != nil {
		r, err = mathSub(ctx.GetLeft().Accept(v), ctx.GetRight().Accept(v))
	}
	if err != nil {
		v.err = err
		return nil
	}
	return r
}

func (v *expVisitor) VisitExpArithmeticMulDivMod(ctx *parser.ExpArithmeticMulDivModContext) interface{} {
	var r interface{}
	var err error
	if ctx.MUL() != nil {
		r, err = mathMul(ctx.GetLeft().Accept(v), ctx.GetRight().Accept(v))
	} else if ctx.DIV() != nil {
		r, err = mathDiv(ctx.GetLeft().Accept(v), ctx.GetRight().Accept(v))
	} else if ctx.MOD() != nil {
		r, err = mathMod(ctx.GetLeft().Accept(v), ctx.GetRight().Accept(v))
	}
	if err != nil {
		v.err = err
		return nil
	}
	return r
}

func (v *expVisitor) VisitExpText(ctx *parser.ExpTextContext) interface{} {
	return toStr(ctx)
}

func (v *expVisitor) VisitExpNumber(ctx *parser.ExpNumberContext) interface{} {
	i, err := toNumber(ctx)
	if err != nil {
		v.err = err
		return nil
	}
	return i
}

func (v *expVisitor) VisitExpFloat(ctx *parser.ExpFloatContext) interface{} {
	i, err := toFloat(ctx)
	if err != nil {
		v.err = err
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
	return ctx.Boolean().Accept(v)
}

func (v *expVisitor) VisitExpFunction(ctx *parser.ExpFunctionContext) interface{} {
	name := ctx.NAME().GetText()
	method, ok := methods[name]
	if !ok {
		v.err = fmt.Errorf("call to unknown function %s", name)
		return nil
	}

	var err error
	var val interface{}
	if ctx.Arguments() != nil {
		args := ctx.Arguments().Accept(v).([]interface{})
		val, err = method(args)
	} else {
		val, err = method(make([]interface{}, 0))
	}

	if err != nil {
		v.err = err
		return false
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
	if ctx.TRUE() != nil {
		return true
	}
	return false
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
	if ctx.VariableExp() != nil {
		return ctx.VariableExp().Accept(v)
	}
	return Null
}

func (v *expVisitor) VisitVariableExp(ctx *parser.VariableExpContext) interface{} {
	for _, entry := range ctx.AllVariable() {
		resolved := entry.Accept(v)
		if resolved != Null {
			return resolved
		}
	}
	return Null
}

func (v *expVisitor) VisitVariable(ctx *parser.VariableContext) interface{} {
	if ctx.Constant() != nil {
		return ctx.Constant().Accept(v)
	}
	if v.vars == nil {
		v.err = ErrMissingVarStore
		return nil
	}

	var name string
	if ctx.NAME() != nil {
		name = ctx.NAME().GetText()
	} else if ctx.VNAME() != nil {
		name = ctx.VNAME().GetText()
	}
	val, ok := v.vars.Lookup(name)
	if !ok {
		return Null
	}
	return val
}

func (v *expVisitor) VisitConstant(ctx *parser.ConstantContext) interface{} {
	if ctx.STEXT() != nil {
		return toStr(ctx.STEXT())
	}
	if ctx.DTEXT() != nil {
		return toStr(ctx.DTEXT())
	}
	if ctx.FLOAT() != nil {
		i, err := toFloat(ctx.FLOAT())
		if err != nil {
			v.err = err
			return nil
		}
		return i
	}
	if ctx.NUMBER() != nil {
		i, err := toNumber(ctx.NUMBER())
		if err != nil {
			v.err = err
			return nil
		}
		return i
	}
	if ctx.Boolean() != nil {
		return ctx.Boolean().Accept(v)
	}
	return nil
}

func (v *expVisitor) VisitExpArray(ctx *parser.ExpArrayContext) interface{} {
	if ctx.Array() != nil {
		return ctx.Array().Accept(v).([]interface{})
	}
	return make([]interface{}, 0)
}

func (v *expVisitor) VisitArray(ctx *parser.ArrayContext) interface{} {
	var args []interface{}

	for _, val := range ctx.AllConstant() {
		args = append(args, val.Accept(v))
	}

	return args
}

func (v *expVisitor) VisitExpDict(ctx *parser.ExpDictContext) interface{} {
	if ctx.Dict() != nil {
		return ctx.Dict().Accept(v).(map[string]interface{})
	}
	return map[string]interface{}{}
}

func (v *expVisitor) VisitKey(ctx *parser.KeyContext) interface{} {
	var key string
	if ctx.STEXT() != nil {
		key = toStr(ctx.STEXT())
	}
	if ctx.DTEXT() != nil {
		key = toStr(ctx.DTEXT())
	}
	if ctx.NAME() != nil {
		key = ctx.NAME().GetText()
	}
	return dictKey{key, ctx.Constant().Accept(v)}
}

func (v *expVisitor) VisitDict(ctx *parser.DictContext) interface{} {
	dict := map[string]interface{}{}

	for _, key := range ctx.AllKey() {
		kv := key.Accept(v).(dictKey)
		dict[kv.key] = kv.value
	}

	return dict
}

type parserContext interface {
	GetText() string
}

func toStr(ctx parserContext) string {
	s := ctx.GetText()
	if len(s) <= 2 {
		return ""
	}
	return s[1 : len(s)-1]
}

func toNumber(ctx parserContext) (int, error) {
	i, err := strconv.Atoi(ctx.GetText())
	if err != nil {
		return 0, fmt.Errorf("could not convert %s to an integer", ctx.GetText())
	}
	return i, nil
}

func toFloat(ctx parserContext) (float64, error) {
	i, err := strconv.ParseFloat(ctx.GetText(), 64)
	if err != nil {
		return i, fmt.Errorf("could not convert %s to a float", ctx.GetText())
	}
	return i, nil
}

type dictKey struct {
	key   string
	value interface{}
}

// ensure interface is implemented
var _ parser.EqlVisitor = (*expVisitor)(nil)
