// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Code generated from Boolexp.g4 by ANTLR 4.7.2. DO NOT EDIT.

package parser // Boolexp

import "github.com/antlr/antlr4/runtime/Go/antlr"

// A complete Visitor for a parse tree produced by BoolexpParser.
type BoolexpVisitor interface {
	antlr.ParseTreeVisitor

	// Visit a parse tree produced by BoolexpParser#expList.
	VisitExpList(ctx *ExpListContext) interface{}

	// Visit a parse tree produced by BoolexpParser#ExpArithmeticNEQ.
	VisitExpArithmeticNEQ(ctx *ExpArithmeticNEQContext) interface{}

	// Visit a parse tree produced by BoolexpParser#ExpArithmeticEQ.
	VisitExpArithmeticEQ(ctx *ExpArithmeticEQContext) interface{}

	// Visit a parse tree produced by BoolexpParser#ExpArithmeticGTE.
	VisitExpArithmeticGTE(ctx *ExpArithmeticGTEContext) interface{}

	// Visit a parse tree produced by BoolexpParser#ExpArithmeticLTE.
	VisitExpArithmeticLTE(ctx *ExpArithmeticLTEContext) interface{}

	// Visit a parse tree produced by BoolexpParser#ExpArithmeticGT.
	VisitExpArithmeticGT(ctx *ExpArithmeticGTContext) interface{}

	// Visit a parse tree produced by BoolexpParser#ExpText.
	VisitExpText(ctx *ExpTextContext) interface{}

	// Visit a parse tree produced by BoolexpParser#ExpNumber.
	VisitExpNumber(ctx *ExpNumberContext) interface{}

	// Visit a parse tree produced by BoolexpParser#ExpLogicalAnd.
	VisitExpLogicalAnd(ctx *ExpLogicalAndContext) interface{}

	// Visit a parse tree produced by BoolexpParser#ExpLogicalOR.
	VisitExpLogicalOR(ctx *ExpLogicalORContext) interface{}

	// Visit a parse tree produced by BoolexpParser#ExpFloat.
	VisitExpFloat(ctx *ExpFloatContext) interface{}

	// Visit a parse tree produced by BoolexpParser#ExpVariable.
	VisitExpVariable(ctx *ExpVariableContext) interface{}

	// Visit a parse tree produced by BoolexpParser#ExpNot.
	VisitExpNot(ctx *ExpNotContext) interface{}

	// Visit a parse tree produced by BoolexpParser#ExpInParen.
	VisitExpInParen(ctx *ExpInParenContext) interface{}

	// Visit a parse tree produced by BoolexpParser#ExpBoolean.
	VisitExpBoolean(ctx *ExpBooleanContext) interface{}

	// Visit a parse tree produced by BoolexpParser#ExpFunction.
	VisitExpFunction(ctx *ExpFunctionContext) interface{}

	// Visit a parse tree produced by BoolexpParser#ExpArithmeticLT.
	VisitExpArithmeticLT(ctx *ExpArithmeticLTContext) interface{}

	// Visit a parse tree produced by BoolexpParser#boolean.
	VisitBoolean(ctx *BooleanContext) interface{}

	// Visit a parse tree produced by BoolexpParser#arguments.
	VisitArguments(ctx *ArgumentsContext) interface{}
}
