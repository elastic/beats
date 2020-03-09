// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Code generated from Boolexp.g4 by ANTLR 4.7.2. DO NOT EDIT.

package parser // Boolexp

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/antlr/antlr4/runtime/Go/antlr"
)

// Suppress unused import errors
var _ = fmt.Printf
var _ = reflect.Copy
var _ = strconv.Itoa

var parserATN = []uint16{
	3, 24715, 42794, 33075, 47597, 16764, 15335, 30598, 22884, 3, 22, 73, 4,
	2, 9, 2, 4, 3, 9, 3, 4, 4, 9, 4, 4, 5, 9, 5, 3, 2, 3, 2, 3, 2, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 5, 3, 26,
	10, 3, 3, 3, 3, 3, 3, 3, 3, 3, 5, 3, 32, 10, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 7, 3, 58, 10, 3, 12, 3,
	14, 3, 61, 11, 3, 3, 4, 3, 4, 3, 5, 3, 5, 3, 5, 7, 5, 68, 10, 5, 12, 5,
	14, 5, 71, 11, 5, 3, 5, 2, 3, 4, 6, 2, 4, 6, 8, 2, 3, 3, 2, 12, 13, 2,
	85, 2, 10, 3, 2, 2, 2, 4, 31, 3, 2, 2, 2, 6, 62, 3, 2, 2, 2, 8, 64, 3,
	2, 2, 2, 10, 11, 5, 4, 3, 2, 11, 12, 7, 2, 2, 3, 12, 3, 3, 2, 2, 2, 13,
	14, 8, 3, 1, 2, 14, 15, 7, 21, 2, 2, 15, 16, 5, 4, 3, 2, 16, 17, 7, 22,
	2, 2, 17, 32, 3, 2, 2, 2, 18, 19, 7, 17, 2, 2, 19, 32, 5, 4, 3, 17, 20,
	32, 5, 6, 4, 2, 21, 32, 7, 18, 2, 2, 22, 23, 7, 19, 2, 2, 23, 25, 7, 21,
	2, 2, 24, 26, 5, 8, 5, 2, 25, 24, 3, 2, 2, 2, 25, 26, 3, 2, 2, 2, 26, 27,
	3, 2, 2, 2, 27, 32, 7, 22, 2, 2, 28, 32, 7, 20, 2, 2, 29, 32, 7, 14, 2,
	2, 30, 32, 7, 15, 2, 2, 31, 13, 3, 2, 2, 2, 31, 18, 3, 2, 2, 2, 31, 20,
	3, 2, 2, 2, 31, 21, 3, 2, 2, 2, 31, 22, 3, 2, 2, 2, 31, 28, 3, 2, 2, 2,
	31, 29, 3, 2, 2, 2, 31, 30, 3, 2, 2, 2, 32, 59, 3, 2, 2, 2, 33, 34, 12,
	16, 2, 2, 34, 35, 7, 4, 2, 2, 35, 58, 5, 4, 3, 17, 36, 37, 12, 15, 2, 2,
	37, 38, 7, 5, 2, 2, 38, 58, 5, 4, 3, 16, 39, 40, 12, 14, 2, 2, 40, 41,
	7, 9, 2, 2, 41, 58, 5, 4, 3, 15, 42, 43, 12, 13, 2, 2, 43, 44, 7, 8, 2,
	2, 44, 58, 5, 4, 3, 14, 45, 46, 12, 12, 2, 2, 46, 47, 7, 7, 2, 2, 47, 58,
	5, 4, 3, 13, 48, 49, 12, 11, 2, 2, 49, 50, 7, 6, 2, 2, 50, 58, 5, 4, 3,
	12, 51, 52, 12, 10, 2, 2, 52, 53, 7, 10, 2, 2, 53, 58, 5, 4, 3, 11, 54,
	55, 12, 9, 2, 2, 55, 56, 7, 11, 2, 2, 56, 58, 5, 4, 3, 10, 57, 33, 3, 2,
	2, 2, 57, 36, 3, 2, 2, 2, 57, 39, 3, 2, 2, 2, 57, 42, 3, 2, 2, 2, 57, 45,
	3, 2, 2, 2, 57, 48, 3, 2, 2, 2, 57, 51, 3, 2, 2, 2, 57, 54, 3, 2, 2, 2,
	58, 61, 3, 2, 2, 2, 59, 57, 3, 2, 2, 2, 59, 60, 3, 2, 2, 2, 60, 5, 3, 2,
	2, 2, 61, 59, 3, 2, 2, 2, 62, 63, 9, 2, 2, 2, 63, 7, 3, 2, 2, 2, 64, 69,
	5, 4, 3, 2, 65, 66, 7, 3, 2, 2, 66, 68, 5, 4, 3, 2, 67, 65, 3, 2, 2, 2,
	68, 71, 3, 2, 2, 2, 69, 67, 3, 2, 2, 2, 69, 70, 3, 2, 2, 2, 70, 9, 3, 2,
	2, 2, 71, 69, 3, 2, 2, 2, 7, 25, 31, 57, 59, 69,
}
var deserializer = antlr.NewATNDeserializer(nil)
var deserializedATN = deserializer.DeserializeFromUInt16(parserATN)

var literalNames = []string{
	"", "','", "'=='", "'!='", "'>'", "'<'", "'>='", "'<='", "", "", "", "",
	"", "", "", "", "", "", "", "'('", "')'",
}
var symbolicNames = []string{
	"", "", "EQ", "NEQ", "GT", "LT", "GTE", "LTE", "AND", "OR", "TRUE", "FALSE",
	"FLOAT", "NUMBER", "WHITESPACE", "NOT", "VARIABLE", "METHODNAME", "TEXT",
	"LPAR", "RPAR",
}

var ruleNames = []string{
	"expList", "exp", "boolean", "arguments",
}
var decisionToDFA = make([]*antlr.DFA, len(deserializedATN.DecisionToState))

func init() {
	for index, ds := range deserializedATN.DecisionToState {
		decisionToDFA[index] = antlr.NewDFA(ds, index)
	}
}

type BoolexpParser struct {
	*antlr.BaseParser
}

func NewBoolexpParser(input antlr.TokenStream) *BoolexpParser {
	this := new(BoolexpParser)

	this.BaseParser = antlr.NewBaseParser(input)

	this.Interpreter = antlr.NewParserATNSimulator(this, deserializedATN, decisionToDFA, antlr.NewPredictionContextCache())
	this.RuleNames = ruleNames
	this.LiteralNames = literalNames
	this.SymbolicNames = symbolicNames
	this.GrammarFileName = "Boolexp.g4"

	return this
}

// BoolexpParser tokens.
const (
	BoolexpParserEOF        = antlr.TokenEOF
	BoolexpParserT__0       = 1
	BoolexpParserEQ         = 2
	BoolexpParserNEQ        = 3
	BoolexpParserGT         = 4
	BoolexpParserLT         = 5
	BoolexpParserGTE        = 6
	BoolexpParserLTE        = 7
	BoolexpParserAND        = 8
	BoolexpParserOR         = 9
	BoolexpParserTRUE       = 10
	BoolexpParserFALSE      = 11
	BoolexpParserFLOAT      = 12
	BoolexpParserNUMBER     = 13
	BoolexpParserWHITESPACE = 14
	BoolexpParserNOT        = 15
	BoolexpParserVARIABLE   = 16
	BoolexpParserMETHODNAME = 17
	BoolexpParserTEXT       = 18
	BoolexpParserLPAR       = 19
	BoolexpParserRPAR       = 20
)

// BoolexpParser rules.
const (
	BoolexpParserRULE_expList   = 0
	BoolexpParserRULE_exp       = 1
	BoolexpParserRULE_boolean   = 2
	BoolexpParserRULE_arguments = 3
)

// IExpListContext is an interface to support dynamic dispatch.
type IExpListContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// IsExpListContext differentiates from other interfaces.
	IsExpListContext()
}

type ExpListContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyExpListContext() *ExpListContext {
	var p = new(ExpListContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = BoolexpParserRULE_expList
	return p
}

func (*ExpListContext) IsExpListContext() {}

func NewExpListContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExpListContext {
	var p = new(ExpListContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = BoolexpParserRULE_expList

	return p
}

func (s *ExpListContext) GetParser() antlr.Parser { return s.parser }

func (s *ExpListContext) Exp() IExpContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IExpContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(IExpContext)
}

func (s *ExpListContext) EOF() antlr.TerminalNode {
	return s.GetToken(BoolexpParserEOF, 0)
}

func (s *ExpListContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpListContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ExpListContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.EnterExpList(s)
	}
}

func (s *ExpListContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.ExitExpList(s)
	}
}

func (s *ExpListContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case BoolexpVisitor:
		return t.VisitExpList(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *BoolexpParser) ExpList() (localctx IExpListContext) {
	localctx = NewExpListContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 0, BoolexpParserRULE_expList)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(8)
		p.exp(0)
	}
	{
		p.SetState(9)
		p.Match(BoolexpParserEOF)
	}

	return localctx
}

// IExpContext is an interface to support dynamic dispatch.
type IExpContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// IsExpContext differentiates from other interfaces.
	IsExpContext()
}

type ExpContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyExpContext() *ExpContext {
	var p = new(ExpContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = BoolexpParserRULE_exp
	return p
}

func (*ExpContext) IsExpContext() {}

func NewExpContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExpContext {
	var p = new(ExpContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = BoolexpParserRULE_exp

	return p
}

func (s *ExpContext) GetParser() antlr.Parser { return s.parser }

func (s *ExpContext) CopyFrom(ctx *ExpContext) {
	s.BaseParserRuleContext.CopyFrom(ctx.BaseParserRuleContext)
}

func (s *ExpContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

type ExpArithmeticNEQContext struct {
	*ExpContext
	left  IExpContext
	right IExpContext
}

func NewExpArithmeticNEQContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *ExpArithmeticNEQContext {
	var p = new(ExpArithmeticNEQContext)

	p.ExpContext = NewEmptyExpContext()
	p.parser = parser
	p.CopyFrom(ctx.(*ExpContext))

	return p
}

func (s *ExpArithmeticNEQContext) GetLeft() IExpContext { return s.left }

func (s *ExpArithmeticNEQContext) GetRight() IExpContext { return s.right }

func (s *ExpArithmeticNEQContext) SetLeft(v IExpContext) { s.left = v }

func (s *ExpArithmeticNEQContext) SetRight(v IExpContext) { s.right = v }

func (s *ExpArithmeticNEQContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpArithmeticNEQContext) NEQ() antlr.TerminalNode {
	return s.GetToken(BoolexpParserNEQ, 0)
}

func (s *ExpArithmeticNEQContext) AllExp() []IExpContext {
	var ts = s.GetTypedRuleContexts(reflect.TypeOf((*IExpContext)(nil)).Elem())
	var tst = make([]IExpContext, len(ts))

	for i, t := range ts {
		if t != nil {
			tst[i] = t.(IExpContext)
		}
	}

	return tst
}

func (s *ExpArithmeticNEQContext) Exp(i int) IExpContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IExpContext)(nil)).Elem(), i)

	if t == nil {
		return nil
	}

	return t.(IExpContext)
}

func (s *ExpArithmeticNEQContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.EnterExpArithmeticNEQ(s)
	}
}

func (s *ExpArithmeticNEQContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.ExitExpArithmeticNEQ(s)
	}
}

func (s *ExpArithmeticNEQContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case BoolexpVisitor:
		return t.VisitExpArithmeticNEQ(s)

	default:
		return t.VisitChildren(s)
	}
}

type ExpArithmeticEQContext struct {
	*ExpContext
	left  IExpContext
	right IExpContext
}

func NewExpArithmeticEQContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *ExpArithmeticEQContext {
	var p = new(ExpArithmeticEQContext)

	p.ExpContext = NewEmptyExpContext()
	p.parser = parser
	p.CopyFrom(ctx.(*ExpContext))

	return p
}

func (s *ExpArithmeticEQContext) GetLeft() IExpContext { return s.left }

func (s *ExpArithmeticEQContext) GetRight() IExpContext { return s.right }

func (s *ExpArithmeticEQContext) SetLeft(v IExpContext) { s.left = v }

func (s *ExpArithmeticEQContext) SetRight(v IExpContext) { s.right = v }

func (s *ExpArithmeticEQContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpArithmeticEQContext) EQ() antlr.TerminalNode {
	return s.GetToken(BoolexpParserEQ, 0)
}

func (s *ExpArithmeticEQContext) AllExp() []IExpContext {
	var ts = s.GetTypedRuleContexts(reflect.TypeOf((*IExpContext)(nil)).Elem())
	var tst = make([]IExpContext, len(ts))

	for i, t := range ts {
		if t != nil {
			tst[i] = t.(IExpContext)
		}
	}

	return tst
}

func (s *ExpArithmeticEQContext) Exp(i int) IExpContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IExpContext)(nil)).Elem(), i)

	if t == nil {
		return nil
	}

	return t.(IExpContext)
}

func (s *ExpArithmeticEQContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.EnterExpArithmeticEQ(s)
	}
}

func (s *ExpArithmeticEQContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.ExitExpArithmeticEQ(s)
	}
}

func (s *ExpArithmeticEQContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case BoolexpVisitor:
		return t.VisitExpArithmeticEQ(s)

	default:
		return t.VisitChildren(s)
	}
}

type ExpArithmeticGTEContext struct {
	*ExpContext
	left  IExpContext
	right IExpContext
}

func NewExpArithmeticGTEContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *ExpArithmeticGTEContext {
	var p = new(ExpArithmeticGTEContext)

	p.ExpContext = NewEmptyExpContext()
	p.parser = parser
	p.CopyFrom(ctx.(*ExpContext))

	return p
}

func (s *ExpArithmeticGTEContext) GetLeft() IExpContext { return s.left }

func (s *ExpArithmeticGTEContext) GetRight() IExpContext { return s.right }

func (s *ExpArithmeticGTEContext) SetLeft(v IExpContext) { s.left = v }

func (s *ExpArithmeticGTEContext) SetRight(v IExpContext) { s.right = v }

func (s *ExpArithmeticGTEContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpArithmeticGTEContext) GTE() antlr.TerminalNode {
	return s.GetToken(BoolexpParserGTE, 0)
}

func (s *ExpArithmeticGTEContext) AllExp() []IExpContext {
	var ts = s.GetTypedRuleContexts(reflect.TypeOf((*IExpContext)(nil)).Elem())
	var tst = make([]IExpContext, len(ts))

	for i, t := range ts {
		if t != nil {
			tst[i] = t.(IExpContext)
		}
	}

	return tst
}

func (s *ExpArithmeticGTEContext) Exp(i int) IExpContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IExpContext)(nil)).Elem(), i)

	if t == nil {
		return nil
	}

	return t.(IExpContext)
}

func (s *ExpArithmeticGTEContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.EnterExpArithmeticGTE(s)
	}
}

func (s *ExpArithmeticGTEContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.ExitExpArithmeticGTE(s)
	}
}

func (s *ExpArithmeticGTEContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case BoolexpVisitor:
		return t.VisitExpArithmeticGTE(s)

	default:
		return t.VisitChildren(s)
	}
}

type ExpArithmeticLTEContext struct {
	*ExpContext
	left  IExpContext
	right IExpContext
}

func NewExpArithmeticLTEContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *ExpArithmeticLTEContext {
	var p = new(ExpArithmeticLTEContext)

	p.ExpContext = NewEmptyExpContext()
	p.parser = parser
	p.CopyFrom(ctx.(*ExpContext))

	return p
}

func (s *ExpArithmeticLTEContext) GetLeft() IExpContext { return s.left }

func (s *ExpArithmeticLTEContext) GetRight() IExpContext { return s.right }

func (s *ExpArithmeticLTEContext) SetLeft(v IExpContext) { s.left = v }

func (s *ExpArithmeticLTEContext) SetRight(v IExpContext) { s.right = v }

func (s *ExpArithmeticLTEContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpArithmeticLTEContext) LTE() antlr.TerminalNode {
	return s.GetToken(BoolexpParserLTE, 0)
}

func (s *ExpArithmeticLTEContext) AllExp() []IExpContext {
	var ts = s.GetTypedRuleContexts(reflect.TypeOf((*IExpContext)(nil)).Elem())
	var tst = make([]IExpContext, len(ts))

	for i, t := range ts {
		if t != nil {
			tst[i] = t.(IExpContext)
		}
	}

	return tst
}

func (s *ExpArithmeticLTEContext) Exp(i int) IExpContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IExpContext)(nil)).Elem(), i)

	if t == nil {
		return nil
	}

	return t.(IExpContext)
}

func (s *ExpArithmeticLTEContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.EnterExpArithmeticLTE(s)
	}
}

func (s *ExpArithmeticLTEContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.ExitExpArithmeticLTE(s)
	}
}

func (s *ExpArithmeticLTEContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case BoolexpVisitor:
		return t.VisitExpArithmeticLTE(s)

	default:
		return t.VisitChildren(s)
	}
}

type ExpArithmeticGTContext struct {
	*ExpContext
	left  IExpContext
	right IExpContext
}

func NewExpArithmeticGTContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *ExpArithmeticGTContext {
	var p = new(ExpArithmeticGTContext)

	p.ExpContext = NewEmptyExpContext()
	p.parser = parser
	p.CopyFrom(ctx.(*ExpContext))

	return p
}

func (s *ExpArithmeticGTContext) GetLeft() IExpContext { return s.left }

func (s *ExpArithmeticGTContext) GetRight() IExpContext { return s.right }

func (s *ExpArithmeticGTContext) SetLeft(v IExpContext) { s.left = v }

func (s *ExpArithmeticGTContext) SetRight(v IExpContext) { s.right = v }

func (s *ExpArithmeticGTContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpArithmeticGTContext) GT() antlr.TerminalNode {
	return s.GetToken(BoolexpParserGT, 0)
}

func (s *ExpArithmeticGTContext) AllExp() []IExpContext {
	var ts = s.GetTypedRuleContexts(reflect.TypeOf((*IExpContext)(nil)).Elem())
	var tst = make([]IExpContext, len(ts))

	for i, t := range ts {
		if t != nil {
			tst[i] = t.(IExpContext)
		}
	}

	return tst
}

func (s *ExpArithmeticGTContext) Exp(i int) IExpContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IExpContext)(nil)).Elem(), i)

	if t == nil {
		return nil
	}

	return t.(IExpContext)
}

func (s *ExpArithmeticGTContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.EnterExpArithmeticGT(s)
	}
}

func (s *ExpArithmeticGTContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.ExitExpArithmeticGT(s)
	}
}

func (s *ExpArithmeticGTContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case BoolexpVisitor:
		return t.VisitExpArithmeticGT(s)

	default:
		return t.VisitChildren(s)
	}
}

type ExpTextContext struct {
	*ExpContext
}

func NewExpTextContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *ExpTextContext {
	var p = new(ExpTextContext)

	p.ExpContext = NewEmptyExpContext()
	p.parser = parser
	p.CopyFrom(ctx.(*ExpContext))

	return p
}

func (s *ExpTextContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpTextContext) TEXT() antlr.TerminalNode {
	return s.GetToken(BoolexpParserTEXT, 0)
}

func (s *ExpTextContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.EnterExpText(s)
	}
}

func (s *ExpTextContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.ExitExpText(s)
	}
}

func (s *ExpTextContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case BoolexpVisitor:
		return t.VisitExpText(s)

	default:
		return t.VisitChildren(s)
	}
}

type ExpNumberContext struct {
	*ExpContext
}

func NewExpNumberContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *ExpNumberContext {
	var p = new(ExpNumberContext)

	p.ExpContext = NewEmptyExpContext()
	p.parser = parser
	p.CopyFrom(ctx.(*ExpContext))

	return p
}

func (s *ExpNumberContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpNumberContext) NUMBER() antlr.TerminalNode {
	return s.GetToken(BoolexpParserNUMBER, 0)
}

func (s *ExpNumberContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.EnterExpNumber(s)
	}
}

func (s *ExpNumberContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.ExitExpNumber(s)
	}
}

func (s *ExpNumberContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case BoolexpVisitor:
		return t.VisitExpNumber(s)

	default:
		return t.VisitChildren(s)
	}
}

type ExpLogicalAndContext struct {
	*ExpContext
	left  IExpContext
	right IExpContext
}

func NewExpLogicalAndContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *ExpLogicalAndContext {
	var p = new(ExpLogicalAndContext)

	p.ExpContext = NewEmptyExpContext()
	p.parser = parser
	p.CopyFrom(ctx.(*ExpContext))

	return p
}

func (s *ExpLogicalAndContext) GetLeft() IExpContext { return s.left }

func (s *ExpLogicalAndContext) GetRight() IExpContext { return s.right }

func (s *ExpLogicalAndContext) SetLeft(v IExpContext) { s.left = v }

func (s *ExpLogicalAndContext) SetRight(v IExpContext) { s.right = v }

func (s *ExpLogicalAndContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpLogicalAndContext) AND() antlr.TerminalNode {
	return s.GetToken(BoolexpParserAND, 0)
}

func (s *ExpLogicalAndContext) AllExp() []IExpContext {
	var ts = s.GetTypedRuleContexts(reflect.TypeOf((*IExpContext)(nil)).Elem())
	var tst = make([]IExpContext, len(ts))

	for i, t := range ts {
		if t != nil {
			tst[i] = t.(IExpContext)
		}
	}

	return tst
}

func (s *ExpLogicalAndContext) Exp(i int) IExpContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IExpContext)(nil)).Elem(), i)

	if t == nil {
		return nil
	}

	return t.(IExpContext)
}

func (s *ExpLogicalAndContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.EnterExpLogicalAnd(s)
	}
}

func (s *ExpLogicalAndContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.ExitExpLogicalAnd(s)
	}
}

func (s *ExpLogicalAndContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case BoolexpVisitor:
		return t.VisitExpLogicalAnd(s)

	default:
		return t.VisitChildren(s)
	}
}

type ExpLogicalORContext struct {
	*ExpContext
	left  IExpContext
	right IExpContext
}

func NewExpLogicalORContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *ExpLogicalORContext {
	var p = new(ExpLogicalORContext)

	p.ExpContext = NewEmptyExpContext()
	p.parser = parser
	p.CopyFrom(ctx.(*ExpContext))

	return p
}

func (s *ExpLogicalORContext) GetLeft() IExpContext { return s.left }

func (s *ExpLogicalORContext) GetRight() IExpContext { return s.right }

func (s *ExpLogicalORContext) SetLeft(v IExpContext) { s.left = v }

func (s *ExpLogicalORContext) SetRight(v IExpContext) { s.right = v }

func (s *ExpLogicalORContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpLogicalORContext) OR() antlr.TerminalNode {
	return s.GetToken(BoolexpParserOR, 0)
}

func (s *ExpLogicalORContext) AllExp() []IExpContext {
	var ts = s.GetTypedRuleContexts(reflect.TypeOf((*IExpContext)(nil)).Elem())
	var tst = make([]IExpContext, len(ts))

	for i, t := range ts {
		if t != nil {
			tst[i] = t.(IExpContext)
		}
	}

	return tst
}

func (s *ExpLogicalORContext) Exp(i int) IExpContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IExpContext)(nil)).Elem(), i)

	if t == nil {
		return nil
	}

	return t.(IExpContext)
}

func (s *ExpLogicalORContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.EnterExpLogicalOR(s)
	}
}

func (s *ExpLogicalORContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.ExitExpLogicalOR(s)
	}
}

func (s *ExpLogicalORContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case BoolexpVisitor:
		return t.VisitExpLogicalOR(s)

	default:
		return t.VisitChildren(s)
	}
}

type ExpFloatContext struct {
	*ExpContext
}

func NewExpFloatContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *ExpFloatContext {
	var p = new(ExpFloatContext)

	p.ExpContext = NewEmptyExpContext()
	p.parser = parser
	p.CopyFrom(ctx.(*ExpContext))

	return p
}

func (s *ExpFloatContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpFloatContext) FLOAT() antlr.TerminalNode {
	return s.GetToken(BoolexpParserFLOAT, 0)
}

func (s *ExpFloatContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.EnterExpFloat(s)
	}
}

func (s *ExpFloatContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.ExitExpFloat(s)
	}
}

func (s *ExpFloatContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case BoolexpVisitor:
		return t.VisitExpFloat(s)

	default:
		return t.VisitChildren(s)
	}
}

type ExpVariableContext struct {
	*ExpContext
}

func NewExpVariableContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *ExpVariableContext {
	var p = new(ExpVariableContext)

	p.ExpContext = NewEmptyExpContext()
	p.parser = parser
	p.CopyFrom(ctx.(*ExpContext))

	return p
}

func (s *ExpVariableContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpVariableContext) VARIABLE() antlr.TerminalNode {
	return s.GetToken(BoolexpParserVARIABLE, 0)
}

func (s *ExpVariableContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.EnterExpVariable(s)
	}
}

func (s *ExpVariableContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.ExitExpVariable(s)
	}
}

func (s *ExpVariableContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case BoolexpVisitor:
		return t.VisitExpVariable(s)

	default:
		return t.VisitChildren(s)
	}
}

type ExpNotContext struct {
	*ExpContext
}

func NewExpNotContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *ExpNotContext {
	var p = new(ExpNotContext)

	p.ExpContext = NewEmptyExpContext()
	p.parser = parser
	p.CopyFrom(ctx.(*ExpContext))

	return p
}

func (s *ExpNotContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpNotContext) NOT() antlr.TerminalNode {
	return s.GetToken(BoolexpParserNOT, 0)
}

func (s *ExpNotContext) Exp() IExpContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IExpContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(IExpContext)
}

func (s *ExpNotContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.EnterExpNot(s)
	}
}

func (s *ExpNotContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.ExitExpNot(s)
	}
}

func (s *ExpNotContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case BoolexpVisitor:
		return t.VisitExpNot(s)

	default:
		return t.VisitChildren(s)
	}
}

type ExpInParenContext struct {
	*ExpContext
}

func NewExpInParenContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *ExpInParenContext {
	var p = new(ExpInParenContext)

	p.ExpContext = NewEmptyExpContext()
	p.parser = parser
	p.CopyFrom(ctx.(*ExpContext))

	return p
}

func (s *ExpInParenContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpInParenContext) LPAR() antlr.TerminalNode {
	return s.GetToken(BoolexpParserLPAR, 0)
}

func (s *ExpInParenContext) Exp() IExpContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IExpContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(IExpContext)
}

func (s *ExpInParenContext) RPAR() antlr.TerminalNode {
	return s.GetToken(BoolexpParserRPAR, 0)
}

func (s *ExpInParenContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.EnterExpInParen(s)
	}
}

func (s *ExpInParenContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.ExitExpInParen(s)
	}
}

func (s *ExpInParenContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case BoolexpVisitor:
		return t.VisitExpInParen(s)

	default:
		return t.VisitChildren(s)
	}
}

type ExpBooleanContext struct {
	*ExpContext
}

func NewExpBooleanContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *ExpBooleanContext {
	var p = new(ExpBooleanContext)

	p.ExpContext = NewEmptyExpContext()
	p.parser = parser
	p.CopyFrom(ctx.(*ExpContext))

	return p
}

func (s *ExpBooleanContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpBooleanContext) Boolean() IBooleanContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IBooleanContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(IBooleanContext)
}

func (s *ExpBooleanContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.EnterExpBoolean(s)
	}
}

func (s *ExpBooleanContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.ExitExpBoolean(s)
	}
}

func (s *ExpBooleanContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case BoolexpVisitor:
		return t.VisitExpBoolean(s)

	default:
		return t.VisitChildren(s)
	}
}

type ExpFunctionContext struct {
	*ExpContext
}

func NewExpFunctionContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *ExpFunctionContext {
	var p = new(ExpFunctionContext)

	p.ExpContext = NewEmptyExpContext()
	p.parser = parser
	p.CopyFrom(ctx.(*ExpContext))

	return p
}

func (s *ExpFunctionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpFunctionContext) METHODNAME() antlr.TerminalNode {
	return s.GetToken(BoolexpParserMETHODNAME, 0)
}

func (s *ExpFunctionContext) LPAR() antlr.TerminalNode {
	return s.GetToken(BoolexpParserLPAR, 0)
}

func (s *ExpFunctionContext) RPAR() antlr.TerminalNode {
	return s.GetToken(BoolexpParserRPAR, 0)
}

func (s *ExpFunctionContext) Arguments() IArgumentsContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IArgumentsContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(IArgumentsContext)
}

func (s *ExpFunctionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.EnterExpFunction(s)
	}
}

func (s *ExpFunctionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.ExitExpFunction(s)
	}
}

func (s *ExpFunctionContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case BoolexpVisitor:
		return t.VisitExpFunction(s)

	default:
		return t.VisitChildren(s)
	}
}

type ExpArithmeticLTContext struct {
	*ExpContext
	left  IExpContext
	right IExpContext
}

func NewExpArithmeticLTContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *ExpArithmeticLTContext {
	var p = new(ExpArithmeticLTContext)

	p.ExpContext = NewEmptyExpContext()
	p.parser = parser
	p.CopyFrom(ctx.(*ExpContext))

	return p
}

func (s *ExpArithmeticLTContext) GetLeft() IExpContext { return s.left }

func (s *ExpArithmeticLTContext) GetRight() IExpContext { return s.right }

func (s *ExpArithmeticLTContext) SetLeft(v IExpContext) { s.left = v }

func (s *ExpArithmeticLTContext) SetRight(v IExpContext) { s.right = v }

func (s *ExpArithmeticLTContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpArithmeticLTContext) LT() antlr.TerminalNode {
	return s.GetToken(BoolexpParserLT, 0)
}

func (s *ExpArithmeticLTContext) AllExp() []IExpContext {
	var ts = s.GetTypedRuleContexts(reflect.TypeOf((*IExpContext)(nil)).Elem())
	var tst = make([]IExpContext, len(ts))

	for i, t := range ts {
		if t != nil {
			tst[i] = t.(IExpContext)
		}
	}

	return tst
}

func (s *ExpArithmeticLTContext) Exp(i int) IExpContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IExpContext)(nil)).Elem(), i)

	if t == nil {
		return nil
	}

	return t.(IExpContext)
}

func (s *ExpArithmeticLTContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.EnterExpArithmeticLT(s)
	}
}

func (s *ExpArithmeticLTContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.ExitExpArithmeticLT(s)
	}
}

func (s *ExpArithmeticLTContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case BoolexpVisitor:
		return t.VisitExpArithmeticLT(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *BoolexpParser) Exp() (localctx IExpContext) {
	return p.exp(0)
}

func (p *BoolexpParser) exp(_p int) (localctx IExpContext) {
	var _parentctx antlr.ParserRuleContext = p.GetParserRuleContext()
	_parentState := p.GetState()
	localctx = NewExpContext(p, p.GetParserRuleContext(), _parentState)
	var _prevctx IExpContext = localctx
	var _ antlr.ParserRuleContext = _prevctx // TODO: To prevent unused variable warning.
	_startState := 2
	p.EnterRecursionRule(localctx, 2, BoolexpParserRULE_exp, _p)
	var _la int

	defer func() {
		p.UnrollRecursionContexts(_parentctx)
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(29)
	p.GetErrorHandler().Sync(p)

	switch p.GetTokenStream().LA(1) {
	case BoolexpParserLPAR:
		localctx = NewExpInParenContext(p, localctx)
		p.SetParserRuleContext(localctx)
		_prevctx = localctx

		{
			p.SetState(12)
			p.Match(BoolexpParserLPAR)
		}
		{
			p.SetState(13)
			p.exp(0)
		}
		{
			p.SetState(14)
			p.Match(BoolexpParserRPAR)
		}

	case BoolexpParserNOT:
		localctx = NewExpNotContext(p, localctx)
		p.SetParserRuleContext(localctx)
		_prevctx = localctx
		{
			p.SetState(16)
			p.Match(BoolexpParserNOT)
		}
		{
			p.SetState(17)
			p.exp(15)
		}

	case BoolexpParserTRUE, BoolexpParserFALSE:
		localctx = NewExpBooleanContext(p, localctx)
		p.SetParserRuleContext(localctx)
		_prevctx = localctx
		{
			p.SetState(18)
			p.Boolean()
		}

	case BoolexpParserVARIABLE:
		localctx = NewExpVariableContext(p, localctx)
		p.SetParserRuleContext(localctx)
		_prevctx = localctx
		{
			p.SetState(19)
			p.Match(BoolexpParserVARIABLE)
		}

	case BoolexpParserMETHODNAME:
		localctx = NewExpFunctionContext(p, localctx)
		p.SetParserRuleContext(localctx)
		_prevctx = localctx
		{
			p.SetState(20)
			p.Match(BoolexpParserMETHODNAME)
		}
		{
			p.SetState(21)
			p.Match(BoolexpParserLPAR)
		}
		p.SetState(23)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)

		if ((_la)&-(0x1f+1)) == 0 && ((1<<uint(_la))&((1<<BoolexpParserTRUE)|(1<<BoolexpParserFALSE)|(1<<BoolexpParserFLOAT)|(1<<BoolexpParserNUMBER)|(1<<BoolexpParserNOT)|(1<<BoolexpParserVARIABLE)|(1<<BoolexpParserMETHODNAME)|(1<<BoolexpParserTEXT)|(1<<BoolexpParserLPAR))) != 0 {
			{
				p.SetState(22)
				p.Arguments()
			}

		}
		{
			p.SetState(25)
			p.Match(BoolexpParserRPAR)
		}

	case BoolexpParserTEXT:
		localctx = NewExpTextContext(p, localctx)
		p.SetParserRuleContext(localctx)
		_prevctx = localctx
		{
			p.SetState(26)
			p.Match(BoolexpParserTEXT)
		}

	case BoolexpParserFLOAT:
		localctx = NewExpFloatContext(p, localctx)
		p.SetParserRuleContext(localctx)
		_prevctx = localctx
		{
			p.SetState(27)
			p.Match(BoolexpParserFLOAT)
		}

	case BoolexpParserNUMBER:
		localctx = NewExpNumberContext(p, localctx)
		p.SetParserRuleContext(localctx)
		_prevctx = localctx
		{
			p.SetState(28)
			p.Match(BoolexpParserNUMBER)
		}

	default:
		panic(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
	}
	p.GetParserRuleContext().SetStop(p.GetTokenStream().LT(-1))
	p.SetState(57)
	p.GetErrorHandler().Sync(p)
	_alt = p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 3, p.GetParserRuleContext())

	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			if p.GetParseListeners() != nil {
				p.TriggerExitRuleEvent()
			}
			_prevctx = localctx
			p.SetState(55)
			p.GetErrorHandler().Sync(p)
			switch p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 2, p.GetParserRuleContext()) {
			case 1:
				localctx = NewExpArithmeticEQContext(p, NewExpContext(p, _parentctx, _parentState))
				localctx.(*ExpArithmeticEQContext).left = _prevctx

				p.PushNewRecursionContext(localctx, _startState, BoolexpParserRULE_exp)
				p.SetState(31)

				if !(p.Precpred(p.GetParserRuleContext(), 14)) {
					panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 14)", ""))
				}
				{
					p.SetState(32)
					p.Match(BoolexpParserEQ)
				}
				{
					p.SetState(33)

					var _x = p.exp(15)

					localctx.(*ExpArithmeticEQContext).right = _x
				}

			case 2:
				localctx = NewExpArithmeticNEQContext(p, NewExpContext(p, _parentctx, _parentState))
				localctx.(*ExpArithmeticNEQContext).left = _prevctx

				p.PushNewRecursionContext(localctx, _startState, BoolexpParserRULE_exp)
				p.SetState(34)

				if !(p.Precpred(p.GetParserRuleContext(), 13)) {
					panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 13)", ""))
				}
				{
					p.SetState(35)
					p.Match(BoolexpParserNEQ)
				}
				{
					p.SetState(36)

					var _x = p.exp(14)

					localctx.(*ExpArithmeticNEQContext).right = _x
				}

			case 3:
				localctx = NewExpArithmeticLTEContext(p, NewExpContext(p, _parentctx, _parentState))
				localctx.(*ExpArithmeticLTEContext).left = _prevctx

				p.PushNewRecursionContext(localctx, _startState, BoolexpParserRULE_exp)
				p.SetState(37)

				if !(p.Precpred(p.GetParserRuleContext(), 12)) {
					panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 12)", ""))
				}
				{
					p.SetState(38)
					p.Match(BoolexpParserLTE)
				}
				{
					p.SetState(39)

					var _x = p.exp(13)

					localctx.(*ExpArithmeticLTEContext).right = _x
				}

			case 4:
				localctx = NewExpArithmeticGTEContext(p, NewExpContext(p, _parentctx, _parentState))
				localctx.(*ExpArithmeticGTEContext).left = _prevctx

				p.PushNewRecursionContext(localctx, _startState, BoolexpParserRULE_exp)
				p.SetState(40)

				if !(p.Precpred(p.GetParserRuleContext(), 11)) {
					panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 11)", ""))
				}
				{
					p.SetState(41)
					p.Match(BoolexpParserGTE)
				}
				{
					p.SetState(42)

					var _x = p.exp(12)

					localctx.(*ExpArithmeticGTEContext).right = _x
				}

			case 5:
				localctx = NewExpArithmeticLTContext(p, NewExpContext(p, _parentctx, _parentState))
				localctx.(*ExpArithmeticLTContext).left = _prevctx

				p.PushNewRecursionContext(localctx, _startState, BoolexpParserRULE_exp)
				p.SetState(43)

				if !(p.Precpred(p.GetParserRuleContext(), 10)) {
					panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 10)", ""))
				}
				{
					p.SetState(44)
					p.Match(BoolexpParserLT)
				}
				{
					p.SetState(45)

					var _x = p.exp(11)

					localctx.(*ExpArithmeticLTContext).right = _x
				}

			case 6:
				localctx = NewExpArithmeticGTContext(p, NewExpContext(p, _parentctx, _parentState))
				localctx.(*ExpArithmeticGTContext).left = _prevctx

				p.PushNewRecursionContext(localctx, _startState, BoolexpParserRULE_exp)
				p.SetState(46)

				if !(p.Precpred(p.GetParserRuleContext(), 9)) {
					panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 9)", ""))
				}
				{
					p.SetState(47)
					p.Match(BoolexpParserGT)
				}
				{
					p.SetState(48)

					var _x = p.exp(10)

					localctx.(*ExpArithmeticGTContext).right = _x
				}

			case 7:
				localctx = NewExpLogicalAndContext(p, NewExpContext(p, _parentctx, _parentState))
				localctx.(*ExpLogicalAndContext).left = _prevctx

				p.PushNewRecursionContext(localctx, _startState, BoolexpParserRULE_exp)
				p.SetState(49)

				if !(p.Precpred(p.GetParserRuleContext(), 8)) {
					panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 8)", ""))
				}
				{
					p.SetState(50)
					p.Match(BoolexpParserAND)
				}
				{
					p.SetState(51)

					var _x = p.exp(9)

					localctx.(*ExpLogicalAndContext).right = _x
				}

			case 8:
				localctx = NewExpLogicalORContext(p, NewExpContext(p, _parentctx, _parentState))
				localctx.(*ExpLogicalORContext).left = _prevctx

				p.PushNewRecursionContext(localctx, _startState, BoolexpParserRULE_exp)
				p.SetState(52)

				if !(p.Precpred(p.GetParserRuleContext(), 7)) {
					panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 7)", ""))
				}
				{
					p.SetState(53)
					p.Match(BoolexpParserOR)
				}
				{
					p.SetState(54)

					var _x = p.exp(8)

					localctx.(*ExpLogicalORContext).right = _x
				}

			}

		}
		p.SetState(59)
		p.GetErrorHandler().Sync(p)
		_alt = p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 3, p.GetParserRuleContext())
	}

	return localctx
}

// IBooleanContext is an interface to support dynamic dispatch.
type IBooleanContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// IsBooleanContext differentiates from other interfaces.
	IsBooleanContext()
}

type BooleanContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyBooleanContext() *BooleanContext {
	var p = new(BooleanContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = BoolexpParserRULE_boolean
	return p
}

func (*BooleanContext) IsBooleanContext() {}

func NewBooleanContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BooleanContext {
	var p = new(BooleanContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = BoolexpParserRULE_boolean

	return p
}

func (s *BooleanContext) GetParser() antlr.Parser { return s.parser }

func (s *BooleanContext) TRUE() antlr.TerminalNode {
	return s.GetToken(BoolexpParserTRUE, 0)
}

func (s *BooleanContext) FALSE() antlr.TerminalNode {
	return s.GetToken(BoolexpParserFALSE, 0)
}

func (s *BooleanContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BooleanContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BooleanContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.EnterBoolean(s)
	}
}

func (s *BooleanContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.ExitBoolean(s)
	}
}

func (s *BooleanContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case BoolexpVisitor:
		return t.VisitBoolean(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *BoolexpParser) Boolean() (localctx IBooleanContext) {
	localctx = NewBooleanContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 4, BoolexpParserRULE_boolean)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(60)
		_la = p.GetTokenStream().LA(1)

		if !(_la == BoolexpParserTRUE || _la == BoolexpParserFALSE) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

	return localctx
}

// IArgumentsContext is an interface to support dynamic dispatch.
type IArgumentsContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// IsArgumentsContext differentiates from other interfaces.
	IsArgumentsContext()
}

type ArgumentsContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyArgumentsContext() *ArgumentsContext {
	var p = new(ArgumentsContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = BoolexpParserRULE_arguments
	return p
}

func (*ArgumentsContext) IsArgumentsContext() {}

func NewArgumentsContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ArgumentsContext {
	var p = new(ArgumentsContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = BoolexpParserRULE_arguments

	return p
}

func (s *ArgumentsContext) GetParser() antlr.Parser { return s.parser }

func (s *ArgumentsContext) AllExp() []IExpContext {
	var ts = s.GetTypedRuleContexts(reflect.TypeOf((*IExpContext)(nil)).Elem())
	var tst = make([]IExpContext, len(ts))

	for i, t := range ts {
		if t != nil {
			tst[i] = t.(IExpContext)
		}
	}

	return tst
}

func (s *ArgumentsContext) Exp(i int) IExpContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IExpContext)(nil)).Elem(), i)

	if t == nil {
		return nil
	}

	return t.(IExpContext)
}

func (s *ArgumentsContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ArgumentsContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ArgumentsContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.EnterArguments(s)
	}
}

func (s *ArgumentsContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(BoolexpListener); ok {
		listenerT.ExitArguments(s)
	}
}

func (s *ArgumentsContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case BoolexpVisitor:
		return t.VisitArguments(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *BoolexpParser) Arguments() (localctx IArgumentsContext) {
	localctx = NewArgumentsContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 6, BoolexpParserRULE_arguments)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(62)
		p.exp(0)
	}
	p.SetState(67)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	for _la == BoolexpParserT__0 {
		{
			p.SetState(63)
			p.Match(BoolexpParserT__0)
		}
		{
			p.SetState(64)
			p.exp(0)
		}

		p.SetState(69)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)
	}

	return localctx
}

func (p *BoolexpParser) Sempred(localctx antlr.RuleContext, ruleIndex, predIndex int) bool {
	switch ruleIndex {
	case 1:
		var t *ExpContext = nil
		if localctx != nil {
			t = localctx.(*ExpContext)
		}
		return p.Exp_Sempred(t, predIndex)

	default:
		panic("No predicate with index: " + fmt.Sprint(ruleIndex))
	}
}

func (p *BoolexpParser) Exp_Sempred(localctx antlr.RuleContext, predIndex int) bool {
	switch predIndex {
	case 0:
		return p.Precpred(p.GetParserRuleContext(), 14)

	case 1:
		return p.Precpred(p.GetParserRuleContext(), 13)

	case 2:
		return p.Precpred(p.GetParserRuleContext(), 12)

	case 3:
		return p.Precpred(p.GetParserRuleContext(), 11)

	case 4:
		return p.Precpred(p.GetParserRuleContext(), 10)

	case 5:
		return p.Precpred(p.GetParserRuleContext(), 9)

	case 6:
		return p.Precpred(p.GetParserRuleContext(), 8)

	case 7:
		return p.Precpred(p.GetParserRuleContext(), 7)

	default:
		panic("No predicate with index: " + fmt.Sprint(predIndex))
	}
}
