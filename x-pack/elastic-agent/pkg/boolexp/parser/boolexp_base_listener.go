// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Code generated from Boolexp.g4 by ANTLR 4.7.2. DO NOT EDIT.

package parser // Boolexp

import "github.com/antlr/antlr4/runtime/Go/antlr"

// BaseBoolexpListener is a complete listener for a parse tree produced by BoolexpParser.
type BaseBoolexpListener struct{}

var _ BoolexpListener = &BaseBoolexpListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseBoolexpListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseBoolexpListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseBoolexpListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseBoolexpListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterExpList is called when production expList is entered.
func (s *BaseBoolexpListener) EnterExpList(ctx *ExpListContext) {}

// ExitExpList is called when production expList is exited.
func (s *BaseBoolexpListener) ExitExpList(ctx *ExpListContext) {}

// EnterExpArithmeticNEQ is called when production ExpArithmeticNEQ is entered.
func (s *BaseBoolexpListener) EnterExpArithmeticNEQ(ctx *ExpArithmeticNEQContext) {}

// ExitExpArithmeticNEQ is called when production ExpArithmeticNEQ is exited.
func (s *BaseBoolexpListener) ExitExpArithmeticNEQ(ctx *ExpArithmeticNEQContext) {}

// EnterExpArithmeticEQ is called when production ExpArithmeticEQ is entered.
func (s *BaseBoolexpListener) EnterExpArithmeticEQ(ctx *ExpArithmeticEQContext) {}

// ExitExpArithmeticEQ is called when production ExpArithmeticEQ is exited.
func (s *BaseBoolexpListener) ExitExpArithmeticEQ(ctx *ExpArithmeticEQContext) {}

// EnterExpArithmeticGTE is called when production ExpArithmeticGTE is entered.
func (s *BaseBoolexpListener) EnterExpArithmeticGTE(ctx *ExpArithmeticGTEContext) {}

// ExitExpArithmeticGTE is called when production ExpArithmeticGTE is exited.
func (s *BaseBoolexpListener) ExitExpArithmeticGTE(ctx *ExpArithmeticGTEContext) {}

// EnterExpArithmeticLTE is called when production ExpArithmeticLTE is entered.
func (s *BaseBoolexpListener) EnterExpArithmeticLTE(ctx *ExpArithmeticLTEContext) {}

// ExitExpArithmeticLTE is called when production ExpArithmeticLTE is exited.
func (s *BaseBoolexpListener) ExitExpArithmeticLTE(ctx *ExpArithmeticLTEContext) {}

// EnterExpArithmeticGT is called when production ExpArithmeticGT is entered.
func (s *BaseBoolexpListener) EnterExpArithmeticGT(ctx *ExpArithmeticGTContext) {}

// ExitExpArithmeticGT is called when production ExpArithmeticGT is exited.
func (s *BaseBoolexpListener) ExitExpArithmeticGT(ctx *ExpArithmeticGTContext) {}

// EnterExpText is called when production ExpText is entered.
func (s *BaseBoolexpListener) EnterExpText(ctx *ExpTextContext) {}

// ExitExpText is called when production ExpText is exited.
func (s *BaseBoolexpListener) ExitExpText(ctx *ExpTextContext) {}

// EnterExpNumber is called when production ExpNumber is entered.
func (s *BaseBoolexpListener) EnterExpNumber(ctx *ExpNumberContext) {}

// ExitExpNumber is called when production ExpNumber is exited.
func (s *BaseBoolexpListener) ExitExpNumber(ctx *ExpNumberContext) {}

// EnterExpLogicalAnd is called when production ExpLogicalAnd is entered.
func (s *BaseBoolexpListener) EnterExpLogicalAnd(ctx *ExpLogicalAndContext) {}

// ExitExpLogicalAnd is called when production ExpLogicalAnd is exited.
func (s *BaseBoolexpListener) ExitExpLogicalAnd(ctx *ExpLogicalAndContext) {}

// EnterExpLogicalOR is called when production ExpLogicalOR is entered.
func (s *BaseBoolexpListener) EnterExpLogicalOR(ctx *ExpLogicalORContext) {}

// ExitExpLogicalOR is called when production ExpLogicalOR is exited.
func (s *BaseBoolexpListener) ExitExpLogicalOR(ctx *ExpLogicalORContext) {}

// EnterExpFloat is called when production ExpFloat is entered.
func (s *BaseBoolexpListener) EnterExpFloat(ctx *ExpFloatContext) {}

// ExitExpFloat is called when production ExpFloat is exited.
func (s *BaseBoolexpListener) ExitExpFloat(ctx *ExpFloatContext) {}

// EnterExpVariable is called when production ExpVariable is entered.
func (s *BaseBoolexpListener) EnterExpVariable(ctx *ExpVariableContext) {}

// ExitExpVariable is called when production ExpVariable is exited.
func (s *BaseBoolexpListener) ExitExpVariable(ctx *ExpVariableContext) {}

// EnterExpNot is called when production ExpNot is entered.
func (s *BaseBoolexpListener) EnterExpNot(ctx *ExpNotContext) {}

// ExitExpNot is called when production ExpNot is exited.
func (s *BaseBoolexpListener) ExitExpNot(ctx *ExpNotContext) {}

// EnterExpInParen is called when production ExpInParen is entered.
func (s *BaseBoolexpListener) EnterExpInParen(ctx *ExpInParenContext) {}

// ExitExpInParen is called when production ExpInParen is exited.
func (s *BaseBoolexpListener) ExitExpInParen(ctx *ExpInParenContext) {}

// EnterExpBoolean is called when production ExpBoolean is entered.
func (s *BaseBoolexpListener) EnterExpBoolean(ctx *ExpBooleanContext) {}

// ExitExpBoolean is called when production ExpBoolean is exited.
func (s *BaseBoolexpListener) ExitExpBoolean(ctx *ExpBooleanContext) {}

// EnterExpFunction is called when production ExpFunction is entered.
func (s *BaseBoolexpListener) EnterExpFunction(ctx *ExpFunctionContext) {}

// ExitExpFunction is called when production ExpFunction is exited.
func (s *BaseBoolexpListener) ExitExpFunction(ctx *ExpFunctionContext) {}

// EnterExpArithmeticLT is called when production ExpArithmeticLT is entered.
func (s *BaseBoolexpListener) EnterExpArithmeticLT(ctx *ExpArithmeticLTContext) {}

// ExitExpArithmeticLT is called when production ExpArithmeticLT is exited.
func (s *BaseBoolexpListener) ExitExpArithmeticLT(ctx *ExpArithmeticLTContext) {}

// EnterBoolean is called when production boolean is entered.
func (s *BaseBoolexpListener) EnterBoolean(ctx *BooleanContext) {}

// ExitBoolean is called when production boolean is exited.
func (s *BaseBoolexpListener) ExitBoolean(ctx *BooleanContext) {}

// EnterArguments is called when production arguments is entered.
func (s *BaseBoolexpListener) EnterArguments(ctx *ArgumentsContext) {}

// ExitArguments is called when production arguments is exited.
func (s *BaseBoolexpListener) ExitArguments(ctx *ArgumentsContext) {}
