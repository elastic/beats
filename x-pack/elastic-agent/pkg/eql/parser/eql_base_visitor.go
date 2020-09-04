// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Code generated from Eql.g4 by ANTLR 4.7.1. DO NOT EDIT.

package parser // Eql

import "github.com/antlr/antlr4/runtime/Go/antlr"

type BaseEqlVisitor struct {
	*antlr.BaseParseTreeVisitor
}

func (v *BaseEqlVisitor) VisitExpList(ctx *ExpListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitBoolean(ctx *BooleanContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitConstant(ctx *ConstantContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitVariable(ctx *VariableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitVariableExp(ctx *VariableExpContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitExpArithmeticNEQ(ctx *ExpArithmeticNEQContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitExpArithmeticEQ(ctx *ExpArithmeticEQContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitExpArithmeticGTE(ctx *ExpArithmeticGTEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitExpArithmeticLTE(ctx *ExpArithmeticLTEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitExpArithmeticGT(ctx *ExpArithmeticGTContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitExpArithmeticMulDivMod(ctx *ExpArithmeticMulDivModContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitExpDict(ctx *ExpDictContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitExpText(ctx *ExpTextContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitExpNumber(ctx *ExpNumberContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitExpLogicalAnd(ctx *ExpLogicalAndContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitExpLogicalOR(ctx *ExpLogicalORContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitExpFloat(ctx *ExpFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitExpVariable(ctx *ExpVariableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitExpArray(ctx *ExpArrayContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitExpNot(ctx *ExpNotContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitExpInParen(ctx *ExpInParenContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitExpBoolean(ctx *ExpBooleanContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitExpArithmeticAddSub(ctx *ExpArithmeticAddSubContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitExpFunction(ctx *ExpFunctionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitExpArithmeticLT(ctx *ExpArithmeticLTContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitArguments(ctx *ArgumentsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitArray(ctx *ArrayContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitKey(ctx *KeyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseEqlVisitor) VisitDict(ctx *DictContext) interface{} {
	return v.VisitChildren(ctx)
}
