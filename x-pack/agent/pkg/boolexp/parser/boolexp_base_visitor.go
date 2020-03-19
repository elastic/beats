// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Code generated from Boolexp.g4 by ANTLR 4.7.2. DO NOT EDIT.

package parser // Boolexp

import "github.com/antlr/antlr4/runtime/Go/antlr"

type BaseBoolexpVisitor struct {
	*antlr.BaseParseTreeVisitor
}

func (v *BaseBoolexpVisitor) VisitExpList(ctx *ExpListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseBoolexpVisitor) VisitExpArithmeticNEQ(ctx *ExpArithmeticNEQContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseBoolexpVisitor) VisitExpArithmeticEQ(ctx *ExpArithmeticEQContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseBoolexpVisitor) VisitExpArithmeticGTE(ctx *ExpArithmeticGTEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseBoolexpVisitor) VisitExpArithmeticLTE(ctx *ExpArithmeticLTEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseBoolexpVisitor) VisitExpArithmeticGT(ctx *ExpArithmeticGTContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseBoolexpVisitor) VisitExpText(ctx *ExpTextContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseBoolexpVisitor) VisitExpNumber(ctx *ExpNumberContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseBoolexpVisitor) VisitExpLogicalAnd(ctx *ExpLogicalAndContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseBoolexpVisitor) VisitExpLogicalOR(ctx *ExpLogicalORContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseBoolexpVisitor) VisitExpFloat(ctx *ExpFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseBoolexpVisitor) VisitExpVariable(ctx *ExpVariableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseBoolexpVisitor) VisitExpNot(ctx *ExpNotContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseBoolexpVisitor) VisitExpInParen(ctx *ExpInParenContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseBoolexpVisitor) VisitExpBoolean(ctx *ExpBooleanContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseBoolexpVisitor) VisitExpFunction(ctx *ExpFunctionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseBoolexpVisitor) VisitExpArithmeticLT(ctx *ExpArithmeticLTContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseBoolexpVisitor) VisitBoolean(ctx *BooleanContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseBoolexpVisitor) VisitArguments(ctx *ArgumentsContext) interface{} {
	return v.VisitChildren(ctx)
}
