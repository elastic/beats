// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Code generated from Eql.g4 by ANTLR 4.7.1. DO NOT EDIT.

package parser // Eql

import "github.com/antlr/antlr4/runtime/Go/antlr"

// BaseEqlListener is a complete listener for a parse tree produced by EqlParser.
type BaseEqlListener struct{}

var _ EqlListener = &BaseEqlListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseEqlListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseEqlListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseEqlListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseEqlListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterExpList is called when production expList is entered.
func (s *BaseEqlListener) EnterExpList(ctx *ExpListContext) {}

// ExitExpList is called when production expList is exited.
func (s *BaseEqlListener) ExitExpList(ctx *ExpListContext) {}

// EnterBoolean is called when production boolean is entered.
func (s *BaseEqlListener) EnterBoolean(ctx *BooleanContext) {}

// ExitBoolean is called when production boolean is exited.
func (s *BaseEqlListener) ExitBoolean(ctx *BooleanContext) {}

// EnterConstant is called when production constant is entered.
func (s *BaseEqlListener) EnterConstant(ctx *ConstantContext) {}

// ExitConstant is called when production constant is exited.
func (s *BaseEqlListener) ExitConstant(ctx *ConstantContext) {}

// EnterVariable is called when production variable is entered.
func (s *BaseEqlListener) EnterVariable(ctx *VariableContext) {}

// ExitVariable is called when production variable is exited.
func (s *BaseEqlListener) ExitVariable(ctx *VariableContext) {}

// EnterVariableExp is called when production variableExp is entered.
func (s *BaseEqlListener) EnterVariableExp(ctx *VariableExpContext) {}

// ExitVariableExp is called when production variableExp is exited.
func (s *BaseEqlListener) ExitVariableExp(ctx *VariableExpContext) {}

// EnterExpArithmeticNEQ is called when production ExpArithmeticNEQ is entered.
func (s *BaseEqlListener) EnterExpArithmeticNEQ(ctx *ExpArithmeticNEQContext) {}

// ExitExpArithmeticNEQ is called when production ExpArithmeticNEQ is exited.
func (s *BaseEqlListener) ExitExpArithmeticNEQ(ctx *ExpArithmeticNEQContext) {}

// EnterExpArithmeticEQ is called when production ExpArithmeticEQ is entered.
func (s *BaseEqlListener) EnterExpArithmeticEQ(ctx *ExpArithmeticEQContext) {}

// ExitExpArithmeticEQ is called when production ExpArithmeticEQ is exited.
func (s *BaseEqlListener) ExitExpArithmeticEQ(ctx *ExpArithmeticEQContext) {}

// EnterExpArithmeticGTE is called when production ExpArithmeticGTE is entered.
func (s *BaseEqlListener) EnterExpArithmeticGTE(ctx *ExpArithmeticGTEContext) {}

// ExitExpArithmeticGTE is called when production ExpArithmeticGTE is exited.
func (s *BaseEqlListener) ExitExpArithmeticGTE(ctx *ExpArithmeticGTEContext) {}

// EnterExpArithmeticLTE is called when production ExpArithmeticLTE is entered.
func (s *BaseEqlListener) EnterExpArithmeticLTE(ctx *ExpArithmeticLTEContext) {}

// ExitExpArithmeticLTE is called when production ExpArithmeticLTE is exited.
func (s *BaseEqlListener) ExitExpArithmeticLTE(ctx *ExpArithmeticLTEContext) {}

// EnterExpArithmeticGT is called when production ExpArithmeticGT is entered.
func (s *BaseEqlListener) EnterExpArithmeticGT(ctx *ExpArithmeticGTContext) {}

// ExitExpArithmeticGT is called when production ExpArithmeticGT is exited.
func (s *BaseEqlListener) ExitExpArithmeticGT(ctx *ExpArithmeticGTContext) {}

// EnterExpArithmeticMulDivMod is called when production ExpArithmeticMulDivMod is entered.
func (s *BaseEqlListener) EnterExpArithmeticMulDivMod(ctx *ExpArithmeticMulDivModContext) {}

// ExitExpArithmeticMulDivMod is called when production ExpArithmeticMulDivMod is exited.
func (s *BaseEqlListener) ExitExpArithmeticMulDivMod(ctx *ExpArithmeticMulDivModContext) {}

// EnterExpDict is called when production ExpDict is entered.
func (s *BaseEqlListener) EnterExpDict(ctx *ExpDictContext) {}

// ExitExpDict is called when production ExpDict is exited.
func (s *BaseEqlListener) ExitExpDict(ctx *ExpDictContext) {}

// EnterExpText is called when production ExpText is entered.
func (s *BaseEqlListener) EnterExpText(ctx *ExpTextContext) {}

// ExitExpText is called when production ExpText is exited.
func (s *BaseEqlListener) ExitExpText(ctx *ExpTextContext) {}

// EnterExpNumber is called when production ExpNumber is entered.
func (s *BaseEqlListener) EnterExpNumber(ctx *ExpNumberContext) {}

// ExitExpNumber is called when production ExpNumber is exited.
func (s *BaseEqlListener) ExitExpNumber(ctx *ExpNumberContext) {}

// EnterExpLogicalAnd is called when production ExpLogicalAnd is entered.
func (s *BaseEqlListener) EnterExpLogicalAnd(ctx *ExpLogicalAndContext) {}

// ExitExpLogicalAnd is called when production ExpLogicalAnd is exited.
func (s *BaseEqlListener) ExitExpLogicalAnd(ctx *ExpLogicalAndContext) {}

// EnterExpLogicalOR is called when production ExpLogicalOR is entered.
func (s *BaseEqlListener) EnterExpLogicalOR(ctx *ExpLogicalORContext) {}

// ExitExpLogicalOR is called when production ExpLogicalOR is exited.
func (s *BaseEqlListener) ExitExpLogicalOR(ctx *ExpLogicalORContext) {}

// EnterExpFloat is called when production ExpFloat is entered.
func (s *BaseEqlListener) EnterExpFloat(ctx *ExpFloatContext) {}

// ExitExpFloat is called when production ExpFloat is exited.
func (s *BaseEqlListener) ExitExpFloat(ctx *ExpFloatContext) {}

// EnterExpVariable is called when production ExpVariable is entered.
func (s *BaseEqlListener) EnterExpVariable(ctx *ExpVariableContext) {}

// ExitExpVariable is called when production ExpVariable is exited.
func (s *BaseEqlListener) ExitExpVariable(ctx *ExpVariableContext) {}

// EnterExpArray is called when production ExpArray is entered.
func (s *BaseEqlListener) EnterExpArray(ctx *ExpArrayContext) {}

// ExitExpArray is called when production ExpArray is exited.
func (s *BaseEqlListener) ExitExpArray(ctx *ExpArrayContext) {}

// EnterExpNot is called when production ExpNot is entered.
func (s *BaseEqlListener) EnterExpNot(ctx *ExpNotContext) {}

// ExitExpNot is called when production ExpNot is exited.
func (s *BaseEqlListener) ExitExpNot(ctx *ExpNotContext) {}

// EnterExpInParen is called when production ExpInParen is entered.
func (s *BaseEqlListener) EnterExpInParen(ctx *ExpInParenContext) {}

// ExitExpInParen is called when production ExpInParen is exited.
func (s *BaseEqlListener) ExitExpInParen(ctx *ExpInParenContext) {}

// EnterExpBoolean is called when production ExpBoolean is entered.
func (s *BaseEqlListener) EnterExpBoolean(ctx *ExpBooleanContext) {}

// ExitExpBoolean is called when production ExpBoolean is exited.
func (s *BaseEqlListener) ExitExpBoolean(ctx *ExpBooleanContext) {}

// EnterExpArithmeticAddSub is called when production ExpArithmeticAddSub is entered.
func (s *BaseEqlListener) EnterExpArithmeticAddSub(ctx *ExpArithmeticAddSubContext) {}

// ExitExpArithmeticAddSub is called when production ExpArithmeticAddSub is exited.
func (s *BaseEqlListener) ExitExpArithmeticAddSub(ctx *ExpArithmeticAddSubContext) {}

// EnterExpFunction is called when production ExpFunction is entered.
func (s *BaseEqlListener) EnterExpFunction(ctx *ExpFunctionContext) {}

// ExitExpFunction is called when production ExpFunction is exited.
func (s *BaseEqlListener) ExitExpFunction(ctx *ExpFunctionContext) {}

// EnterExpArithmeticLT is called when production ExpArithmeticLT is entered.
func (s *BaseEqlListener) EnterExpArithmeticLT(ctx *ExpArithmeticLTContext) {}

// ExitExpArithmeticLT is called when production ExpArithmeticLT is exited.
func (s *BaseEqlListener) ExitExpArithmeticLT(ctx *ExpArithmeticLTContext) {}

// EnterArguments is called when production arguments is entered.
func (s *BaseEqlListener) EnterArguments(ctx *ArgumentsContext) {}

// ExitArguments is called when production arguments is exited.
func (s *BaseEqlListener) ExitArguments(ctx *ArgumentsContext) {}

// EnterArray is called when production array is entered.
func (s *BaseEqlListener) EnterArray(ctx *ArrayContext) {}

// ExitArray is called when production array is exited.
func (s *BaseEqlListener) ExitArray(ctx *ArrayContext) {}

// EnterKey is called when production key is entered.
func (s *BaseEqlListener) EnterKey(ctx *KeyContext) {}

// ExitKey is called when production key is exited.
func (s *BaseEqlListener) ExitKey(ctx *KeyContext) {}

// EnterDict is called when production dict is entered.
func (s *BaseEqlListener) EnterDict(ctx *DictContext) {}

// ExitDict is called when production dict is exited.
func (s *BaseEqlListener) ExitDict(ctx *DictContext) {}
