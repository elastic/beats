// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Code generated from Eql.g4 by ANTLR 4.7.1. DO NOT EDIT.

package parser // Eql

import "github.com/antlr/antlr4/runtime/Go/antlr"

// A complete Visitor for a parse tree produced by EqlParser.
type EqlVisitor interface {
	antlr.ParseTreeVisitor

	// Visit a parse tree produced by EqlParser#expList.
	VisitExpList(ctx *ExpListContext) interface{}

	// Visit a parse tree produced by EqlParser#boolean.
	VisitBoolean(ctx *BooleanContext) interface{}

	// Visit a parse tree produced by EqlParser#constant.
	VisitConstant(ctx *ConstantContext) interface{}

	// Visit a parse tree produced by EqlParser#variable.
	VisitVariable(ctx *VariableContext) interface{}

	// Visit a parse tree produced by EqlParser#variableExp.
	VisitVariableExp(ctx *VariableExpContext) interface{}

	// Visit a parse tree produced by EqlParser#ExpArithmeticNEQ.
	VisitExpArithmeticNEQ(ctx *ExpArithmeticNEQContext) interface{}

	// Visit a parse tree produced by EqlParser#ExpArithmeticEQ.
	VisitExpArithmeticEQ(ctx *ExpArithmeticEQContext) interface{}

	// Visit a parse tree produced by EqlParser#ExpArithmeticGTE.
	VisitExpArithmeticGTE(ctx *ExpArithmeticGTEContext) interface{}

	// Visit a parse tree produced by EqlParser#ExpArithmeticLTE.
	VisitExpArithmeticLTE(ctx *ExpArithmeticLTEContext) interface{}

	// Visit a parse tree produced by EqlParser#ExpArithmeticGT.
	VisitExpArithmeticGT(ctx *ExpArithmeticGTContext) interface{}

	// Visit a parse tree produced by EqlParser#ExpArithmeticMulDivMod.
	VisitExpArithmeticMulDivMod(ctx *ExpArithmeticMulDivModContext) interface{}

	// Visit a parse tree produced by EqlParser#ExpDict.
	VisitExpDict(ctx *ExpDictContext) interface{}

	// Visit a parse tree produced by EqlParser#ExpText.
	VisitExpText(ctx *ExpTextContext) interface{}

	// Visit a parse tree produced by EqlParser#ExpNumber.
	VisitExpNumber(ctx *ExpNumberContext) interface{}

	// Visit a parse tree produced by EqlParser#ExpLogicalAnd.
	VisitExpLogicalAnd(ctx *ExpLogicalAndContext) interface{}

	// Visit a parse tree produced by EqlParser#ExpLogicalOR.
	VisitExpLogicalOR(ctx *ExpLogicalORContext) interface{}

	// Visit a parse tree produced by EqlParser#ExpFloat.
	VisitExpFloat(ctx *ExpFloatContext) interface{}

	// Visit a parse tree produced by EqlParser#ExpVariable.
	VisitExpVariable(ctx *ExpVariableContext) interface{}

	// Visit a parse tree produced by EqlParser#ExpArray.
	VisitExpArray(ctx *ExpArrayContext) interface{}

	// Visit a parse tree produced by EqlParser#ExpNot.
	VisitExpNot(ctx *ExpNotContext) interface{}

	// Visit a parse tree produced by EqlParser#ExpInParen.
	VisitExpInParen(ctx *ExpInParenContext) interface{}

	// Visit a parse tree produced by EqlParser#ExpBoolean.
	VisitExpBoolean(ctx *ExpBooleanContext) interface{}

	// Visit a parse tree produced by EqlParser#ExpArithmeticAddSub.
	VisitExpArithmeticAddSub(ctx *ExpArithmeticAddSubContext) interface{}

	// Visit a parse tree produced by EqlParser#ExpFunction.
	VisitExpFunction(ctx *ExpFunctionContext) interface{}

	// Visit a parse tree produced by EqlParser#ExpArithmeticLT.
	VisitExpArithmeticLT(ctx *ExpArithmeticLTContext) interface{}

	// Visit a parse tree produced by EqlParser#arguments.
	VisitArguments(ctx *ArgumentsContext) interface{}

	// Visit a parse tree produced by EqlParser#array.
	VisitArray(ctx *ArrayContext) interface{}

	// Visit a parse tree produced by EqlParser#key.
	VisitKey(ctx *KeyContext) interface{}

	// Visit a parse tree produced by EqlParser#dict.
	VisitDict(ctx *DictContext) interface{}
}
