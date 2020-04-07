// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Code generated from Boolexp.g4 by ANTLR 4.7.2. DO NOT EDIT.

package parser // Boolexp

import "github.com/antlr/antlr4/runtime/Go/antlr"

// BoolexpListener is a complete listener for a parse tree produced by BoolexpParser.
type BoolexpListener interface {
	antlr.ParseTreeListener

	// EnterExpList is called when entering the expList production.
	EnterExpList(c *ExpListContext)

	// EnterExpArithmeticNEQ is called when entering the ExpArithmeticNEQ production.
	EnterExpArithmeticNEQ(c *ExpArithmeticNEQContext)

	// EnterExpArithmeticEQ is called when entering the ExpArithmeticEQ production.
	EnterExpArithmeticEQ(c *ExpArithmeticEQContext)

	// EnterExpArithmeticGTE is called when entering the ExpArithmeticGTE production.
	EnterExpArithmeticGTE(c *ExpArithmeticGTEContext)

	// EnterExpArithmeticLTE is called when entering the ExpArithmeticLTE production.
	EnterExpArithmeticLTE(c *ExpArithmeticLTEContext)

	// EnterExpArithmeticGT is called when entering the ExpArithmeticGT production.
	EnterExpArithmeticGT(c *ExpArithmeticGTContext)

	// EnterExpText is called when entering the ExpText production.
	EnterExpText(c *ExpTextContext)

	// EnterExpNumber is called when entering the ExpNumber production.
	EnterExpNumber(c *ExpNumberContext)

	// EnterExpLogicalAnd is called when entering the ExpLogicalAnd production.
	EnterExpLogicalAnd(c *ExpLogicalAndContext)

	// EnterExpLogicalOR is called when entering the ExpLogicalOR production.
	EnterExpLogicalOR(c *ExpLogicalORContext)

	// EnterExpFloat is called when entering the ExpFloat production.
	EnterExpFloat(c *ExpFloatContext)

	// EnterExpVariable is called when entering the ExpVariable production.
	EnterExpVariable(c *ExpVariableContext)

	// EnterExpNot is called when entering the ExpNot production.
	EnterExpNot(c *ExpNotContext)

	// EnterExpInParen is called when entering the ExpInParen production.
	EnterExpInParen(c *ExpInParenContext)

	// EnterExpBoolean is called when entering the ExpBoolean production.
	EnterExpBoolean(c *ExpBooleanContext)

	// EnterExpFunction is called when entering the ExpFunction production.
	EnterExpFunction(c *ExpFunctionContext)

	// EnterExpArithmeticLT is called when entering the ExpArithmeticLT production.
	EnterExpArithmeticLT(c *ExpArithmeticLTContext)

	// EnterBoolean is called when entering the boolean production.
	EnterBoolean(c *BooleanContext)

	// EnterArguments is called when entering the arguments production.
	EnterArguments(c *ArgumentsContext)

	// ExitExpList is called when exiting the expList production.
	ExitExpList(c *ExpListContext)

	// ExitExpArithmeticNEQ is called when exiting the ExpArithmeticNEQ production.
	ExitExpArithmeticNEQ(c *ExpArithmeticNEQContext)

	// ExitExpArithmeticEQ is called when exiting the ExpArithmeticEQ production.
	ExitExpArithmeticEQ(c *ExpArithmeticEQContext)

	// ExitExpArithmeticGTE is called when exiting the ExpArithmeticGTE production.
	ExitExpArithmeticGTE(c *ExpArithmeticGTEContext)

	// ExitExpArithmeticLTE is called when exiting the ExpArithmeticLTE production.
	ExitExpArithmeticLTE(c *ExpArithmeticLTEContext)

	// ExitExpArithmeticGT is called when exiting the ExpArithmeticGT production.
	ExitExpArithmeticGT(c *ExpArithmeticGTContext)

	// ExitExpText is called when exiting the ExpText production.
	ExitExpText(c *ExpTextContext)

	// ExitExpNumber is called when exiting the ExpNumber production.
	ExitExpNumber(c *ExpNumberContext)

	// ExitExpLogicalAnd is called when exiting the ExpLogicalAnd production.
	ExitExpLogicalAnd(c *ExpLogicalAndContext)

	// ExitExpLogicalOR is called when exiting the ExpLogicalOR production.
	ExitExpLogicalOR(c *ExpLogicalORContext)

	// ExitExpFloat is called when exiting the ExpFloat production.
	ExitExpFloat(c *ExpFloatContext)

	// ExitExpVariable is called when exiting the ExpVariable production.
	ExitExpVariable(c *ExpVariableContext)

	// ExitExpNot is called when exiting the ExpNot production.
	ExitExpNot(c *ExpNotContext)

	// ExitExpInParen is called when exiting the ExpInParen production.
	ExitExpInParen(c *ExpInParenContext)

	// ExitExpBoolean is called when exiting the ExpBoolean production.
	ExitExpBoolean(c *ExpBooleanContext)

	// ExitExpFunction is called when exiting the ExpFunction production.
	ExitExpFunction(c *ExpFunctionContext)

	// ExitExpArithmeticLT is called when exiting the ExpArithmeticLT production.
	ExitExpArithmeticLT(c *ExpArithmeticLTContext)

	// ExitBoolean is called when exiting the boolean production.
	ExitBoolean(c *BooleanContext)

	// ExitArguments is called when exiting the arguments production.
	ExitArguments(c *ArgumentsContext)
}
