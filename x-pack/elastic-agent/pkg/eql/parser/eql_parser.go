// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Code generated from Eql.g4 by ANTLR 4.7.1. DO NOT EDIT.

package parser // Eql

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
	3, 24715, 42794, 33075, 47597, 16764, 15335, 30598, 22884, 3, 35, 144,
	4, 2, 9, 2, 4, 3, 9, 3, 4, 4, 9, 4, 4, 5, 9, 5, 4, 6, 9, 6, 4, 7, 9, 7,
	4, 8, 9, 8, 4, 9, 9, 9, 4, 10, 9, 10, 4, 11, 9, 11, 3, 2, 3, 2, 3, 2, 3,
	3, 3, 3, 3, 4, 3, 4, 3, 4, 3, 4, 3, 4, 5, 4, 33, 10, 4, 3, 5, 3, 5, 3,
	5, 5, 5, 38, 10, 5, 3, 6, 3, 6, 3, 6, 7, 6, 43, 10, 6, 12, 6, 14, 6, 46,
	11, 6, 3, 7, 3, 7, 3, 7, 3, 7, 3, 7, 3, 7, 3, 7, 3, 7, 3, 7, 3, 7, 3, 7,
	3, 7, 3, 7, 3, 7, 3, 7, 5, 7, 63, 10, 7, 3, 7, 3, 7, 3, 7, 5, 7, 68, 10,
	7, 3, 7, 3, 7, 3, 7, 5, 7, 73, 10, 7, 3, 7, 3, 7, 3, 7, 3, 7, 5, 7, 79,
	10, 7, 3, 7, 3, 7, 3, 7, 3, 7, 3, 7, 3, 7, 3, 7, 3, 7, 3, 7, 3, 7, 3, 7,
	3, 7, 3, 7, 3, 7, 3, 7, 3, 7, 3, 7, 3, 7, 3, 7, 3, 7, 3, 7, 3, 7, 3, 7,
	3, 7, 3, 7, 3, 7, 3, 7, 3, 7, 3, 7, 3, 7, 7, 7, 111, 10, 7, 12, 7, 14,
	7, 114, 11, 7, 3, 8, 3, 8, 3, 8, 7, 8, 119, 10, 8, 12, 8, 14, 8, 122, 11,
	8, 3, 9, 3, 9, 3, 9, 7, 9, 127, 10, 9, 12, 9, 14, 9, 130, 11, 9, 3, 10,
	3, 10, 3, 10, 3, 10, 3, 11, 3, 11, 3, 11, 7, 11, 139, 10, 11, 12, 11, 14,
	11, 142, 11, 11, 3, 11, 2, 3, 12, 12, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20,
	2, 7, 3, 2, 19, 20, 3, 2, 27, 28, 3, 2, 14, 16, 3, 2, 12, 13, 4, 2, 25,
	25, 27, 28, 2, 165, 2, 22, 3, 2, 2, 2, 4, 25, 3, 2, 2, 2, 6, 32, 3, 2,
	2, 2, 8, 37, 3, 2, 2, 2, 10, 39, 3, 2, 2, 2, 12, 78, 3, 2, 2, 2, 14, 115,
	3, 2, 2, 2, 16, 123, 3, 2, 2, 2, 18, 131, 3, 2, 2, 2, 20, 135, 3, 2, 2,
	2, 22, 23, 5, 12, 7, 2, 23, 24, 7, 2, 2, 3, 24, 3, 3, 2, 2, 2, 25, 26,
	9, 2, 2, 2, 26, 5, 3, 2, 2, 2, 27, 33, 7, 27, 2, 2, 28, 33, 7, 28, 2, 2,
	29, 33, 7, 21, 2, 2, 30, 33, 7, 22, 2, 2, 31, 33, 5, 4, 3, 2, 32, 27, 3,
	2, 2, 2, 32, 28, 3, 2, 2, 2, 32, 29, 3, 2, 2, 2, 32, 30, 3, 2, 2, 2, 32,
	31, 3, 2, 2, 2, 33, 7, 3, 2, 2, 2, 34, 38, 7, 25, 2, 2, 35, 38, 7, 26,
	2, 2, 36, 38, 5, 6, 4, 2, 37, 34, 3, 2, 2, 2, 37, 35, 3, 2, 2, 2, 37, 36,
	3, 2, 2, 2, 38, 9, 3, 2, 2, 2, 39, 44, 5, 8, 5, 2, 40, 41, 7, 3, 2, 2,
	41, 43, 5, 8, 5, 2, 42, 40, 3, 2, 2, 2, 43, 46, 3, 2, 2, 2, 44, 42, 3,
	2, 2, 2, 44, 45, 3, 2, 2, 2, 45, 11, 3, 2, 2, 2, 46, 44, 3, 2, 2, 2, 47,
	48, 8, 7, 1, 2, 48, 49, 7, 29, 2, 2, 49, 50, 5, 12, 7, 2, 50, 51, 7, 30,
	2, 2, 51, 79, 3, 2, 2, 2, 52, 53, 7, 24, 2, 2, 53, 79, 5, 12, 7, 19, 54,
	79, 5, 4, 3, 2, 55, 56, 7, 35, 2, 2, 56, 57, 5, 10, 6, 2, 57, 58, 7, 34,
	2, 2, 58, 79, 3, 2, 2, 2, 59, 60, 7, 25, 2, 2, 60, 62, 7, 29, 2, 2, 61,
	63, 5, 14, 8, 2, 62, 61, 3, 2, 2, 2, 62, 63, 3, 2, 2, 2, 63, 64, 3, 2,
	2, 2, 64, 79, 7, 30, 2, 2, 65, 67, 7, 31, 2, 2, 66, 68, 5, 16, 9, 2, 67,
	66, 3, 2, 2, 2, 67, 68, 3, 2, 2, 2, 68, 69, 3, 2, 2, 2, 69, 79, 7, 32,
	2, 2, 70, 72, 7, 33, 2, 2, 71, 73, 5, 20, 11, 2, 72, 71, 3, 2, 2, 2, 72,
	73, 3, 2, 2, 2, 73, 74, 3, 2, 2, 2, 74, 79, 7, 34, 2, 2, 75, 79, 9, 3,
	2, 2, 76, 79, 7, 21, 2, 2, 77, 79, 7, 22, 2, 2, 78, 47, 3, 2, 2, 2, 78,
	52, 3, 2, 2, 2, 78, 54, 3, 2, 2, 2, 78, 55, 3, 2, 2, 2, 78, 59, 3, 2, 2,
	2, 78, 65, 3, 2, 2, 2, 78, 70, 3, 2, 2, 2, 78, 75, 3, 2, 2, 2, 78, 76,
	3, 2, 2, 2, 78, 77, 3, 2, 2, 2, 79, 112, 3, 2, 2, 2, 80, 81, 12, 21, 2,
	2, 81, 82, 9, 4, 2, 2, 82, 111, 5, 12, 7, 22, 83, 84, 12, 20, 2, 2, 84,
	85, 9, 5, 2, 2, 85, 111, 5, 12, 7, 21, 86, 87, 12, 18, 2, 2, 87, 88, 7,
	6, 2, 2, 88, 111, 5, 12, 7, 19, 89, 90, 12, 17, 2, 2, 90, 91, 7, 7, 2,
	2, 91, 111, 5, 12, 7, 18, 92, 93, 12, 16, 2, 2, 93, 94, 7, 11, 2, 2, 94,
	111, 5, 12, 7, 17, 95, 96, 12, 15, 2, 2, 96, 97, 7, 10, 2, 2, 97, 111,
	5, 12, 7, 16, 98, 99, 12, 14, 2, 2, 99, 100, 7, 9, 2, 2, 100, 111, 5, 12,
	7, 15, 101, 102, 12, 13, 2, 2, 102, 103, 7, 8, 2, 2, 103, 111, 5, 12, 7,
	14, 104, 105, 12, 12, 2, 2, 105, 106, 7, 17, 2, 2, 106, 111, 5, 12, 7,
	13, 107, 108, 12, 11, 2, 2, 108, 109, 7, 18, 2, 2, 109, 111, 5, 12, 7,
	12, 110, 80, 3, 2, 2, 2, 110, 83, 3, 2, 2, 2, 110, 86, 3, 2, 2, 2, 110,
	89, 3, 2, 2, 2, 110, 92, 3, 2, 2, 2, 110, 95, 3, 2, 2, 2, 110, 98, 3, 2,
	2, 2, 110, 101, 3, 2, 2, 2, 110, 104, 3, 2, 2, 2, 110, 107, 3, 2, 2, 2,
	111, 114, 3, 2, 2, 2, 112, 110, 3, 2, 2, 2, 112, 113, 3, 2, 2, 2, 113,
	13, 3, 2, 2, 2, 114, 112, 3, 2, 2, 2, 115, 120, 5, 12, 7, 2, 116, 117,
	7, 4, 2, 2, 117, 119, 5, 12, 7, 2, 118, 116, 3, 2, 2, 2, 119, 122, 3, 2,
	2, 2, 120, 118, 3, 2, 2, 2, 120, 121, 3, 2, 2, 2, 121, 15, 3, 2, 2, 2,
	122, 120, 3, 2, 2, 2, 123, 128, 5, 6, 4, 2, 124, 125, 7, 4, 2, 2, 125,
	127, 5, 6, 4, 2, 126, 124, 3, 2, 2, 2, 127, 130, 3, 2, 2, 2, 128, 126,
	3, 2, 2, 2, 128, 129, 3, 2, 2, 2, 129, 17, 3, 2, 2, 2, 130, 128, 3, 2,
	2, 2, 131, 132, 9, 6, 2, 2, 132, 133, 7, 5, 2, 2, 133, 134, 5, 6, 4, 2,
	134, 19, 3, 2, 2, 2, 135, 140, 5, 18, 10, 2, 136, 137, 7, 4, 2, 2, 137,
	139, 5, 18, 10, 2, 138, 136, 3, 2, 2, 2, 139, 142, 3, 2, 2, 2, 140, 138,
	3, 2, 2, 2, 140, 141, 3, 2, 2, 2, 141, 21, 3, 2, 2, 2, 142, 140, 3, 2,
	2, 2, 14, 32, 37, 44, 62, 67, 72, 78, 110, 112, 120, 128, 140,
}
var deserializer = antlr.NewATNDeserializer(nil)
var deserializedATN = deserializer.DeserializeFromUInt16(parserATN)

var literalNames = []string{
	"", "'|'", "','", "':'", "'=='", "'!='", "'>'", "'<'", "'>='", "'<='",
	"'+'", "'-'", "'*'", "'/'", "'%'", "", "", "", "", "", "", "", "", "",
	"", "", "", "'('", "')'", "'['", "']'", "'{'", "'}'", "'${'",
}
var symbolicNames = []string{
	"", "", "", "", "EQ", "NEQ", "GT", "LT", "GTE", "LTE", "ADD", "SUB", "MUL",
	"DIV", "MOD", "AND", "OR", "TRUE", "FALSE", "FLOAT", "NUMBER", "WHITESPACE",
	"NOT", "NAME", "VNAME", "STEXT", "DTEXT", "LPAR", "RPAR", "LARR", "RARR",
	"LDICT", "RDICT", "BEGIN_VARIABLE",
}

var ruleNames = []string{
	"expList", "boolean", "constant", "variable", "variableExp", "exp", "arguments",
	"array", "key", "dict",
}
var decisionToDFA = make([]*antlr.DFA, len(deserializedATN.DecisionToState))

func init() {
	for index, ds := range deserializedATN.DecisionToState {
		decisionToDFA[index] = antlr.NewDFA(ds, index)
	}
}

type EqlParser struct {
	*antlr.BaseParser
}

func NewEqlParser(input antlr.TokenStream) *EqlParser {
	this := new(EqlParser)

	this.BaseParser = antlr.NewBaseParser(input)

	this.Interpreter = antlr.NewParserATNSimulator(this, deserializedATN, decisionToDFA, antlr.NewPredictionContextCache())
	this.RuleNames = ruleNames
	this.LiteralNames = literalNames
	this.SymbolicNames = symbolicNames
	this.GrammarFileName = "Eql.g4"

	return this
}

// EqlParser tokens.
const (
	EqlParserEOF            = antlr.TokenEOF
	EqlParserT__0           = 1
	EqlParserT__1           = 2
	EqlParserT__2           = 3
	EqlParserEQ             = 4
	EqlParserNEQ            = 5
	EqlParserGT             = 6
	EqlParserLT             = 7
	EqlParserGTE            = 8
	EqlParserLTE            = 9
	EqlParserADD            = 10
	EqlParserSUB            = 11
	EqlParserMUL            = 12
	EqlParserDIV            = 13
	EqlParserMOD            = 14
	EqlParserAND            = 15
	EqlParserOR             = 16
	EqlParserTRUE           = 17
	EqlParserFALSE          = 18
	EqlParserFLOAT          = 19
	EqlParserNUMBER         = 20
	EqlParserWHITESPACE     = 21
	EqlParserNOT            = 22
	EqlParserNAME           = 23
	EqlParserVNAME          = 24
	EqlParserSTEXT          = 25
	EqlParserDTEXT          = 26
	EqlParserLPAR           = 27
	EqlParserRPAR           = 28
	EqlParserLARR           = 29
	EqlParserRARR           = 30
	EqlParserLDICT          = 31
	EqlParserRDICT          = 32
	EqlParserBEGIN_VARIABLE = 33
)

// EqlParser rules.
const (
	EqlParserRULE_expList     = 0
	EqlParserRULE_boolean     = 1
	EqlParserRULE_constant    = 2
	EqlParserRULE_variable    = 3
	EqlParserRULE_variableExp = 4
	EqlParserRULE_exp         = 5
	EqlParserRULE_arguments   = 6
	EqlParserRULE_array       = 7
	EqlParserRULE_key         = 8
	EqlParserRULE_dict        = 9
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
	p.RuleIndex = EqlParserRULE_expList
	return p
}

func (*ExpListContext) IsExpListContext() {}

func NewExpListContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExpListContext {
	var p = new(ExpListContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = EqlParserRULE_expList

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
	return s.GetToken(EqlParserEOF, 0)
}

func (s *ExpListContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpListContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ExpListContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterExpList(s)
	}
}

func (s *ExpListContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitExpList(s)
	}
}

func (s *ExpListContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
		return t.VisitExpList(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *EqlParser) ExpList() (localctx IExpListContext) {
	localctx = NewExpListContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 0, EqlParserRULE_expList)

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
		p.SetState(20)
		p.exp(0)
	}
	{
		p.SetState(21)
		p.Match(EqlParserEOF)
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
	p.RuleIndex = EqlParserRULE_boolean
	return p
}

func (*BooleanContext) IsBooleanContext() {}

func NewBooleanContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BooleanContext {
	var p = new(BooleanContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = EqlParserRULE_boolean

	return p
}

func (s *BooleanContext) GetParser() antlr.Parser { return s.parser }

func (s *BooleanContext) TRUE() antlr.TerminalNode {
	return s.GetToken(EqlParserTRUE, 0)
}

func (s *BooleanContext) FALSE() antlr.TerminalNode {
	return s.GetToken(EqlParserFALSE, 0)
}

func (s *BooleanContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BooleanContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BooleanContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterBoolean(s)
	}
}

func (s *BooleanContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitBoolean(s)
	}
}

func (s *BooleanContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
		return t.VisitBoolean(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *EqlParser) Boolean() (localctx IBooleanContext) {
	localctx = NewBooleanContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 2, EqlParserRULE_boolean)
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
		p.SetState(23)
		_la = p.GetTokenStream().LA(1)

		if !(_la == EqlParserTRUE || _la == EqlParserFALSE) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

	return localctx
}

// IConstantContext is an interface to support dynamic dispatch.
type IConstantContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// IsConstantContext differentiates from other interfaces.
	IsConstantContext()
}

type ConstantContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyConstantContext() *ConstantContext {
	var p = new(ConstantContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = EqlParserRULE_constant
	return p
}

func (*ConstantContext) IsConstantContext() {}

func NewConstantContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ConstantContext {
	var p = new(ConstantContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = EqlParserRULE_constant

	return p
}

func (s *ConstantContext) GetParser() antlr.Parser { return s.parser }

func (s *ConstantContext) STEXT() antlr.TerminalNode {
	return s.GetToken(EqlParserSTEXT, 0)
}

func (s *ConstantContext) DTEXT() antlr.TerminalNode {
	return s.GetToken(EqlParserDTEXT, 0)
}

func (s *ConstantContext) FLOAT() antlr.TerminalNode {
	return s.GetToken(EqlParserFLOAT, 0)
}

func (s *ConstantContext) NUMBER() antlr.TerminalNode {
	return s.GetToken(EqlParserNUMBER, 0)
}

func (s *ConstantContext) Boolean() IBooleanContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IBooleanContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(IBooleanContext)
}

func (s *ConstantContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ConstantContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ConstantContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterConstant(s)
	}
}

func (s *ConstantContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitConstant(s)
	}
}

func (s *ConstantContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
		return t.VisitConstant(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *EqlParser) Constant() (localctx IConstantContext) {
	localctx = NewConstantContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 4, EqlParserRULE_constant)

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

	p.SetState(30)
	p.GetErrorHandler().Sync(p)

	switch p.GetTokenStream().LA(1) {
	case EqlParserSTEXT:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(25)
			p.Match(EqlParserSTEXT)
		}

	case EqlParserDTEXT:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(26)
			p.Match(EqlParserDTEXT)
		}

	case EqlParserFLOAT:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(27)
			p.Match(EqlParserFLOAT)
		}

	case EqlParserNUMBER:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(28)
			p.Match(EqlParserNUMBER)
		}

	case EqlParserTRUE, EqlParserFALSE:
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(29)
			p.Boolean()
		}

	default:
		panic(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
	}

	return localctx
}

// IVariableContext is an interface to support dynamic dispatch.
type IVariableContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// IsVariableContext differentiates from other interfaces.
	IsVariableContext()
}

type VariableContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyVariableContext() *VariableContext {
	var p = new(VariableContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = EqlParserRULE_variable
	return p
}

func (*VariableContext) IsVariableContext() {}

func NewVariableContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *VariableContext {
	var p = new(VariableContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = EqlParserRULE_variable

	return p
}

func (s *VariableContext) GetParser() antlr.Parser { return s.parser }

func (s *VariableContext) NAME() antlr.TerminalNode {
	return s.GetToken(EqlParserNAME, 0)
}

func (s *VariableContext) VNAME() antlr.TerminalNode {
	return s.GetToken(EqlParserVNAME, 0)
}

func (s *VariableContext) Constant() IConstantContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IConstantContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(IConstantContext)
}

func (s *VariableContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *VariableContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *VariableContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterVariable(s)
	}
}

func (s *VariableContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitVariable(s)
	}
}

func (s *VariableContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
		return t.VisitVariable(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *EqlParser) Variable() (localctx IVariableContext) {
	localctx = NewVariableContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 6, EqlParserRULE_variable)

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

	p.SetState(35)
	p.GetErrorHandler().Sync(p)

	switch p.GetTokenStream().LA(1) {
	case EqlParserNAME:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(32)
			p.Match(EqlParserNAME)
		}

	case EqlParserVNAME:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(33)
			p.Match(EqlParserVNAME)
		}

	case EqlParserTRUE, EqlParserFALSE, EqlParserFLOAT, EqlParserNUMBER, EqlParserSTEXT, EqlParserDTEXT:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(34)
			p.Constant()
		}

	default:
		panic(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
	}

	return localctx
}

// IVariableExpContext is an interface to support dynamic dispatch.
type IVariableExpContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// IsVariableExpContext differentiates from other interfaces.
	IsVariableExpContext()
}

type VariableExpContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyVariableExpContext() *VariableExpContext {
	var p = new(VariableExpContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = EqlParserRULE_variableExp
	return p
}

func (*VariableExpContext) IsVariableExpContext() {}

func NewVariableExpContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *VariableExpContext {
	var p = new(VariableExpContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = EqlParserRULE_variableExp

	return p
}

func (s *VariableExpContext) GetParser() antlr.Parser { return s.parser }

func (s *VariableExpContext) AllVariable() []IVariableContext {
	var ts = s.GetTypedRuleContexts(reflect.TypeOf((*IVariableContext)(nil)).Elem())
	var tst = make([]IVariableContext, len(ts))

	for i, t := range ts {
		if t != nil {
			tst[i] = t.(IVariableContext)
		}
	}

	return tst
}

func (s *VariableExpContext) Variable(i int) IVariableContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IVariableContext)(nil)).Elem(), i)

	if t == nil {
		return nil
	}

	return t.(IVariableContext)
}

func (s *VariableExpContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *VariableExpContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *VariableExpContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterVariableExp(s)
	}
}

func (s *VariableExpContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitVariableExp(s)
	}
}

func (s *VariableExpContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
		return t.VisitVariableExp(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *EqlParser) VariableExp() (localctx IVariableExpContext) {
	localctx = NewVariableExpContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 8, EqlParserRULE_variableExp)
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
		p.SetState(37)
		p.Variable()
	}
	p.SetState(42)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	for _la == EqlParserT__0 {
		{
			p.SetState(38)
			p.Match(EqlParserT__0)
		}
		{
			p.SetState(39)
			p.Variable()
		}

		p.SetState(44)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)
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
	p.RuleIndex = EqlParserRULE_exp
	return p
}

func (*ExpContext) IsExpContext() {}

func NewExpContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExpContext {
	var p = new(ExpContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = EqlParserRULE_exp

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
	return s.GetToken(EqlParserNEQ, 0)
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
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterExpArithmeticNEQ(s)
	}
}

func (s *ExpArithmeticNEQContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitExpArithmeticNEQ(s)
	}
}

func (s *ExpArithmeticNEQContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
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
	return s.GetToken(EqlParserEQ, 0)
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
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterExpArithmeticEQ(s)
	}
}

func (s *ExpArithmeticEQContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitExpArithmeticEQ(s)
	}
}

func (s *ExpArithmeticEQContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
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
	return s.GetToken(EqlParserGTE, 0)
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
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterExpArithmeticGTE(s)
	}
}

func (s *ExpArithmeticGTEContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitExpArithmeticGTE(s)
	}
}

func (s *ExpArithmeticGTEContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
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
	return s.GetToken(EqlParserLTE, 0)
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
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterExpArithmeticLTE(s)
	}
}

func (s *ExpArithmeticLTEContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitExpArithmeticLTE(s)
	}
}

func (s *ExpArithmeticLTEContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
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
	return s.GetToken(EqlParserGT, 0)
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
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterExpArithmeticGT(s)
	}
}

func (s *ExpArithmeticGTContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitExpArithmeticGT(s)
	}
}

func (s *ExpArithmeticGTContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
		return t.VisitExpArithmeticGT(s)

	default:
		return t.VisitChildren(s)
	}
}

type ExpArithmeticMulDivModContext struct {
	*ExpContext
	left  IExpContext
	right IExpContext
}

func NewExpArithmeticMulDivModContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *ExpArithmeticMulDivModContext {
	var p = new(ExpArithmeticMulDivModContext)

	p.ExpContext = NewEmptyExpContext()
	p.parser = parser
	p.CopyFrom(ctx.(*ExpContext))

	return p
}

func (s *ExpArithmeticMulDivModContext) GetLeft() IExpContext { return s.left }

func (s *ExpArithmeticMulDivModContext) GetRight() IExpContext { return s.right }

func (s *ExpArithmeticMulDivModContext) SetLeft(v IExpContext) { s.left = v }

func (s *ExpArithmeticMulDivModContext) SetRight(v IExpContext) { s.right = v }

func (s *ExpArithmeticMulDivModContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpArithmeticMulDivModContext) AllExp() []IExpContext {
	var ts = s.GetTypedRuleContexts(reflect.TypeOf((*IExpContext)(nil)).Elem())
	var tst = make([]IExpContext, len(ts))

	for i, t := range ts {
		if t != nil {
			tst[i] = t.(IExpContext)
		}
	}

	return tst
}

func (s *ExpArithmeticMulDivModContext) Exp(i int) IExpContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IExpContext)(nil)).Elem(), i)

	if t == nil {
		return nil
	}

	return t.(IExpContext)
}

func (s *ExpArithmeticMulDivModContext) MUL() antlr.TerminalNode {
	return s.GetToken(EqlParserMUL, 0)
}

func (s *ExpArithmeticMulDivModContext) DIV() antlr.TerminalNode {
	return s.GetToken(EqlParserDIV, 0)
}

func (s *ExpArithmeticMulDivModContext) MOD() antlr.TerminalNode {
	return s.GetToken(EqlParserMOD, 0)
}

func (s *ExpArithmeticMulDivModContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterExpArithmeticMulDivMod(s)
	}
}

func (s *ExpArithmeticMulDivModContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitExpArithmeticMulDivMod(s)
	}
}

func (s *ExpArithmeticMulDivModContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
		return t.VisitExpArithmeticMulDivMod(s)

	default:
		return t.VisitChildren(s)
	}
}

type ExpDictContext struct {
	*ExpContext
}

func NewExpDictContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *ExpDictContext {
	var p = new(ExpDictContext)

	p.ExpContext = NewEmptyExpContext()
	p.parser = parser
	p.CopyFrom(ctx.(*ExpContext))

	return p
}

func (s *ExpDictContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpDictContext) LDICT() antlr.TerminalNode {
	return s.GetToken(EqlParserLDICT, 0)
}

func (s *ExpDictContext) RDICT() antlr.TerminalNode {
	return s.GetToken(EqlParserRDICT, 0)
}

func (s *ExpDictContext) Dict() IDictContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IDictContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(IDictContext)
}

func (s *ExpDictContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterExpDict(s)
	}
}

func (s *ExpDictContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitExpDict(s)
	}
}

func (s *ExpDictContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
		return t.VisitExpDict(s)

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

func (s *ExpTextContext) STEXT() antlr.TerminalNode {
	return s.GetToken(EqlParserSTEXT, 0)
}

func (s *ExpTextContext) DTEXT() antlr.TerminalNode {
	return s.GetToken(EqlParserDTEXT, 0)
}

func (s *ExpTextContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterExpText(s)
	}
}

func (s *ExpTextContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitExpText(s)
	}
}

func (s *ExpTextContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
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
	return s.GetToken(EqlParserNUMBER, 0)
}

func (s *ExpNumberContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterExpNumber(s)
	}
}

func (s *ExpNumberContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitExpNumber(s)
	}
}

func (s *ExpNumberContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
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
	return s.GetToken(EqlParserAND, 0)
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
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterExpLogicalAnd(s)
	}
}

func (s *ExpLogicalAndContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitExpLogicalAnd(s)
	}
}

func (s *ExpLogicalAndContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
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
	return s.GetToken(EqlParserOR, 0)
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
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterExpLogicalOR(s)
	}
}

func (s *ExpLogicalORContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitExpLogicalOR(s)
	}
}

func (s *ExpLogicalORContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
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
	return s.GetToken(EqlParserFLOAT, 0)
}

func (s *ExpFloatContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterExpFloat(s)
	}
}

func (s *ExpFloatContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitExpFloat(s)
	}
}

func (s *ExpFloatContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
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

func (s *ExpVariableContext) BEGIN_VARIABLE() antlr.TerminalNode {
	return s.GetToken(EqlParserBEGIN_VARIABLE, 0)
}

func (s *ExpVariableContext) VariableExp() IVariableExpContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IVariableExpContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(IVariableExpContext)
}

func (s *ExpVariableContext) RDICT() antlr.TerminalNode {
	return s.GetToken(EqlParserRDICT, 0)
}

func (s *ExpVariableContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterExpVariable(s)
	}
}

func (s *ExpVariableContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitExpVariable(s)
	}
}

func (s *ExpVariableContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
		return t.VisitExpVariable(s)

	default:
		return t.VisitChildren(s)
	}
}

type ExpArrayContext struct {
	*ExpContext
}

func NewExpArrayContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *ExpArrayContext {
	var p = new(ExpArrayContext)

	p.ExpContext = NewEmptyExpContext()
	p.parser = parser
	p.CopyFrom(ctx.(*ExpContext))

	return p
}

func (s *ExpArrayContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpArrayContext) LARR() antlr.TerminalNode {
	return s.GetToken(EqlParserLARR, 0)
}

func (s *ExpArrayContext) RARR() antlr.TerminalNode {
	return s.GetToken(EqlParserRARR, 0)
}

func (s *ExpArrayContext) Array() IArrayContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IArrayContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(IArrayContext)
}

func (s *ExpArrayContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterExpArray(s)
	}
}

func (s *ExpArrayContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitExpArray(s)
	}
}

func (s *ExpArrayContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
		return t.VisitExpArray(s)

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
	return s.GetToken(EqlParserNOT, 0)
}

func (s *ExpNotContext) Exp() IExpContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IExpContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(IExpContext)
}

func (s *ExpNotContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterExpNot(s)
	}
}

func (s *ExpNotContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitExpNot(s)
	}
}

func (s *ExpNotContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
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
	return s.GetToken(EqlParserLPAR, 0)
}

func (s *ExpInParenContext) Exp() IExpContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IExpContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(IExpContext)
}

func (s *ExpInParenContext) RPAR() antlr.TerminalNode {
	return s.GetToken(EqlParserRPAR, 0)
}

func (s *ExpInParenContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterExpInParen(s)
	}
}

func (s *ExpInParenContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitExpInParen(s)
	}
}

func (s *ExpInParenContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
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
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterExpBoolean(s)
	}
}

func (s *ExpBooleanContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitExpBoolean(s)
	}
}

func (s *ExpBooleanContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
		return t.VisitExpBoolean(s)

	default:
		return t.VisitChildren(s)
	}
}

type ExpArithmeticAddSubContext struct {
	*ExpContext
	left  IExpContext
	right IExpContext
}

func NewExpArithmeticAddSubContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *ExpArithmeticAddSubContext {
	var p = new(ExpArithmeticAddSubContext)

	p.ExpContext = NewEmptyExpContext()
	p.parser = parser
	p.CopyFrom(ctx.(*ExpContext))

	return p
}

func (s *ExpArithmeticAddSubContext) GetLeft() IExpContext { return s.left }

func (s *ExpArithmeticAddSubContext) GetRight() IExpContext { return s.right }

func (s *ExpArithmeticAddSubContext) SetLeft(v IExpContext) { s.left = v }

func (s *ExpArithmeticAddSubContext) SetRight(v IExpContext) { s.right = v }

func (s *ExpArithmeticAddSubContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpArithmeticAddSubContext) AllExp() []IExpContext {
	var ts = s.GetTypedRuleContexts(reflect.TypeOf((*IExpContext)(nil)).Elem())
	var tst = make([]IExpContext, len(ts))

	for i, t := range ts {
		if t != nil {
			tst[i] = t.(IExpContext)
		}
	}

	return tst
}

func (s *ExpArithmeticAddSubContext) Exp(i int) IExpContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IExpContext)(nil)).Elem(), i)

	if t == nil {
		return nil
	}

	return t.(IExpContext)
}

func (s *ExpArithmeticAddSubContext) ADD() antlr.TerminalNode {
	return s.GetToken(EqlParserADD, 0)
}

func (s *ExpArithmeticAddSubContext) SUB() antlr.TerminalNode {
	return s.GetToken(EqlParserSUB, 0)
}

func (s *ExpArithmeticAddSubContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterExpArithmeticAddSub(s)
	}
}

func (s *ExpArithmeticAddSubContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitExpArithmeticAddSub(s)
	}
}

func (s *ExpArithmeticAddSubContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
		return t.VisitExpArithmeticAddSub(s)

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

func (s *ExpFunctionContext) NAME() antlr.TerminalNode {
	return s.GetToken(EqlParserNAME, 0)
}

func (s *ExpFunctionContext) LPAR() antlr.TerminalNode {
	return s.GetToken(EqlParserLPAR, 0)
}

func (s *ExpFunctionContext) RPAR() antlr.TerminalNode {
	return s.GetToken(EqlParserRPAR, 0)
}

func (s *ExpFunctionContext) Arguments() IArgumentsContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IArgumentsContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(IArgumentsContext)
}

func (s *ExpFunctionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterExpFunction(s)
	}
}

func (s *ExpFunctionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitExpFunction(s)
	}
}

func (s *ExpFunctionContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
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
	return s.GetToken(EqlParserLT, 0)
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
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterExpArithmeticLT(s)
	}
}

func (s *ExpArithmeticLTContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitExpArithmeticLT(s)
	}
}

func (s *ExpArithmeticLTContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
		return t.VisitExpArithmeticLT(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *EqlParser) Exp() (localctx IExpContext) {
	return p.exp(0)
}

func (p *EqlParser) exp(_p int) (localctx IExpContext) {
	var _parentctx antlr.ParserRuleContext = p.GetParserRuleContext()
	_parentState := p.GetState()
	localctx = NewExpContext(p, p.GetParserRuleContext(), _parentState)
	var _prevctx IExpContext = localctx
	var _ antlr.ParserRuleContext = _prevctx // TODO: To prevent unused variable warning.
	_startState := 10
	p.EnterRecursionRule(localctx, 10, EqlParserRULE_exp, _p)
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
	p.SetState(76)
	p.GetErrorHandler().Sync(p)

	switch p.GetTokenStream().LA(1) {
	case EqlParserLPAR:
		localctx = NewExpInParenContext(p, localctx)
		p.SetParserRuleContext(localctx)
		_prevctx = localctx

		{
			p.SetState(46)
			p.Match(EqlParserLPAR)
		}
		{
			p.SetState(47)
			p.exp(0)
		}
		{
			p.SetState(48)
			p.Match(EqlParserRPAR)
		}

	case EqlParserNOT:
		localctx = NewExpNotContext(p, localctx)
		p.SetParserRuleContext(localctx)
		_prevctx = localctx
		{
			p.SetState(50)
			p.Match(EqlParserNOT)
		}
		{
			p.SetState(51)
			p.exp(17)
		}

	case EqlParserTRUE, EqlParserFALSE:
		localctx = NewExpBooleanContext(p, localctx)
		p.SetParserRuleContext(localctx)
		_prevctx = localctx
		{
			p.SetState(52)
			p.Boolean()
		}

	case EqlParserBEGIN_VARIABLE:
		localctx = NewExpVariableContext(p, localctx)
		p.SetParserRuleContext(localctx)
		_prevctx = localctx
		{
			p.SetState(53)
			p.Match(EqlParserBEGIN_VARIABLE)
		}
		{
			p.SetState(54)
			p.VariableExp()
		}
		{
			p.SetState(55)
			p.Match(EqlParserRDICT)
		}

	case EqlParserNAME:
		localctx = NewExpFunctionContext(p, localctx)
		p.SetParserRuleContext(localctx)
		_prevctx = localctx
		{
			p.SetState(57)
			p.Match(EqlParserNAME)
		}
		{
			p.SetState(58)
			p.Match(EqlParserLPAR)
		}
		p.SetState(60)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)

		if ((_la-17)&-(0x1f+1)) == 0 && ((1<<uint((_la-17)))&((1<<(EqlParserTRUE-17))|(1<<(EqlParserFALSE-17))|(1<<(EqlParserFLOAT-17))|(1<<(EqlParserNUMBER-17))|(1<<(EqlParserNOT-17))|(1<<(EqlParserNAME-17))|(1<<(EqlParserSTEXT-17))|(1<<(EqlParserDTEXT-17))|(1<<(EqlParserLPAR-17))|(1<<(EqlParserLARR-17))|(1<<(EqlParserLDICT-17))|(1<<(EqlParserBEGIN_VARIABLE-17)))) != 0 {
			{
				p.SetState(59)
				p.Arguments()
			}

		}
		{
			p.SetState(62)
			p.Match(EqlParserRPAR)
		}

	case EqlParserLARR:
		localctx = NewExpArrayContext(p, localctx)
		p.SetParserRuleContext(localctx)
		_prevctx = localctx
		{
			p.SetState(63)
			p.Match(EqlParserLARR)
		}
		p.SetState(65)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)

		if ((_la)&-(0x1f+1)) == 0 && ((1<<uint(_la))&((1<<EqlParserTRUE)|(1<<EqlParserFALSE)|(1<<EqlParserFLOAT)|(1<<EqlParserNUMBER)|(1<<EqlParserSTEXT)|(1<<EqlParserDTEXT))) != 0 {
			{
				p.SetState(64)
				p.Array()
			}

		}
		{
			p.SetState(67)
			p.Match(EqlParserRARR)
		}

	case EqlParserLDICT:
		localctx = NewExpDictContext(p, localctx)
		p.SetParserRuleContext(localctx)
		_prevctx = localctx
		{
			p.SetState(68)
			p.Match(EqlParserLDICT)
		}
		p.SetState(70)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)

		if ((_la)&-(0x1f+1)) == 0 && ((1<<uint(_la))&((1<<EqlParserNAME)|(1<<EqlParserSTEXT)|(1<<EqlParserDTEXT))) != 0 {
			{
				p.SetState(69)
				p.Dict()
			}

		}
		{
			p.SetState(72)
			p.Match(EqlParserRDICT)
		}

	case EqlParserSTEXT, EqlParserDTEXT:
		localctx = NewExpTextContext(p, localctx)
		p.SetParserRuleContext(localctx)
		_prevctx = localctx
		{
			p.SetState(73)
			_la = p.GetTokenStream().LA(1)

			if !(_la == EqlParserSTEXT || _la == EqlParserDTEXT) {
				p.GetErrorHandler().RecoverInline(p)
			} else {
				p.GetErrorHandler().ReportMatch(p)
				p.Consume()
			}
		}

	case EqlParserFLOAT:
		localctx = NewExpFloatContext(p, localctx)
		p.SetParserRuleContext(localctx)
		_prevctx = localctx
		{
			p.SetState(74)
			p.Match(EqlParserFLOAT)
		}

	case EqlParserNUMBER:
		localctx = NewExpNumberContext(p, localctx)
		p.SetParserRuleContext(localctx)
		_prevctx = localctx
		{
			p.SetState(75)
			p.Match(EqlParserNUMBER)
		}

	default:
		panic(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
	}
	p.GetParserRuleContext().SetStop(p.GetTokenStream().LT(-1))
	p.SetState(110)
	p.GetErrorHandler().Sync(p)
	_alt = p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 8, p.GetParserRuleContext())

	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			if p.GetParseListeners() != nil {
				p.TriggerExitRuleEvent()
			}
			_prevctx = localctx
			p.SetState(108)
			p.GetErrorHandler().Sync(p)
			switch p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 7, p.GetParserRuleContext()) {
			case 1:
				localctx = NewExpArithmeticMulDivModContext(p, NewExpContext(p, _parentctx, _parentState))
				localctx.(*ExpArithmeticMulDivModContext).left = _prevctx

				p.PushNewRecursionContext(localctx, _startState, EqlParserRULE_exp)
				p.SetState(78)

				if !(p.Precpred(p.GetParserRuleContext(), 19)) {
					panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 19)", ""))
				}
				{
					p.SetState(79)
					_la = p.GetTokenStream().LA(1)

					if !(((_la)&-(0x1f+1)) == 0 && ((1<<uint(_la))&((1<<EqlParserMUL)|(1<<EqlParserDIV)|(1<<EqlParserMOD))) != 0) {
						p.GetErrorHandler().RecoverInline(p)
					} else {
						p.GetErrorHandler().ReportMatch(p)
						p.Consume()
					}
				}
				{
					p.SetState(80)

					var _x = p.exp(20)

					localctx.(*ExpArithmeticMulDivModContext).right = _x
				}

			case 2:
				localctx = NewExpArithmeticAddSubContext(p, NewExpContext(p, _parentctx, _parentState))
				localctx.(*ExpArithmeticAddSubContext).left = _prevctx

				p.PushNewRecursionContext(localctx, _startState, EqlParserRULE_exp)
				p.SetState(81)

				if !(p.Precpred(p.GetParserRuleContext(), 18)) {
					panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 18)", ""))
				}
				{
					p.SetState(82)
					_la = p.GetTokenStream().LA(1)

					if !(_la == EqlParserADD || _la == EqlParserSUB) {
						p.GetErrorHandler().RecoverInline(p)
					} else {
						p.GetErrorHandler().ReportMatch(p)
						p.Consume()
					}
				}
				{
					p.SetState(83)

					var _x = p.exp(19)

					localctx.(*ExpArithmeticAddSubContext).right = _x
				}

			case 3:
				localctx = NewExpArithmeticEQContext(p, NewExpContext(p, _parentctx, _parentState))
				localctx.(*ExpArithmeticEQContext).left = _prevctx

				p.PushNewRecursionContext(localctx, _startState, EqlParserRULE_exp)
				p.SetState(84)

				if !(p.Precpred(p.GetParserRuleContext(), 16)) {
					panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 16)", ""))
				}
				{
					p.SetState(85)
					p.Match(EqlParserEQ)
				}
				{
					p.SetState(86)

					var _x = p.exp(17)

					localctx.(*ExpArithmeticEQContext).right = _x
				}

			case 4:
				localctx = NewExpArithmeticNEQContext(p, NewExpContext(p, _parentctx, _parentState))
				localctx.(*ExpArithmeticNEQContext).left = _prevctx

				p.PushNewRecursionContext(localctx, _startState, EqlParserRULE_exp)
				p.SetState(87)

				if !(p.Precpred(p.GetParserRuleContext(), 15)) {
					panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 15)", ""))
				}
				{
					p.SetState(88)
					p.Match(EqlParserNEQ)
				}
				{
					p.SetState(89)

					var _x = p.exp(16)

					localctx.(*ExpArithmeticNEQContext).right = _x
				}

			case 5:
				localctx = NewExpArithmeticLTEContext(p, NewExpContext(p, _parentctx, _parentState))
				localctx.(*ExpArithmeticLTEContext).left = _prevctx

				p.PushNewRecursionContext(localctx, _startState, EqlParserRULE_exp)
				p.SetState(90)

				if !(p.Precpred(p.GetParserRuleContext(), 14)) {
					panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 14)", ""))
				}
				{
					p.SetState(91)
					p.Match(EqlParserLTE)
				}
				{
					p.SetState(92)

					var _x = p.exp(15)

					localctx.(*ExpArithmeticLTEContext).right = _x
				}

			case 6:
				localctx = NewExpArithmeticGTEContext(p, NewExpContext(p, _parentctx, _parentState))
				localctx.(*ExpArithmeticGTEContext).left = _prevctx

				p.PushNewRecursionContext(localctx, _startState, EqlParserRULE_exp)
				p.SetState(93)

				if !(p.Precpred(p.GetParserRuleContext(), 13)) {
					panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 13)", ""))
				}
				{
					p.SetState(94)
					p.Match(EqlParserGTE)
				}
				{
					p.SetState(95)

					var _x = p.exp(14)

					localctx.(*ExpArithmeticGTEContext).right = _x
				}

			case 7:
				localctx = NewExpArithmeticLTContext(p, NewExpContext(p, _parentctx, _parentState))
				localctx.(*ExpArithmeticLTContext).left = _prevctx

				p.PushNewRecursionContext(localctx, _startState, EqlParserRULE_exp)
				p.SetState(96)

				if !(p.Precpred(p.GetParserRuleContext(), 12)) {
					panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 12)", ""))
				}
				{
					p.SetState(97)
					p.Match(EqlParserLT)
				}
				{
					p.SetState(98)

					var _x = p.exp(13)

					localctx.(*ExpArithmeticLTContext).right = _x
				}

			case 8:
				localctx = NewExpArithmeticGTContext(p, NewExpContext(p, _parentctx, _parentState))
				localctx.(*ExpArithmeticGTContext).left = _prevctx

				p.PushNewRecursionContext(localctx, _startState, EqlParserRULE_exp)
				p.SetState(99)

				if !(p.Precpred(p.GetParserRuleContext(), 11)) {
					panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 11)", ""))
				}
				{
					p.SetState(100)
					p.Match(EqlParserGT)
				}
				{
					p.SetState(101)

					var _x = p.exp(12)

					localctx.(*ExpArithmeticGTContext).right = _x
				}

			case 9:
				localctx = NewExpLogicalAndContext(p, NewExpContext(p, _parentctx, _parentState))
				localctx.(*ExpLogicalAndContext).left = _prevctx

				p.PushNewRecursionContext(localctx, _startState, EqlParserRULE_exp)
				p.SetState(102)

				if !(p.Precpred(p.GetParserRuleContext(), 10)) {
					panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 10)", ""))
				}
				{
					p.SetState(103)
					p.Match(EqlParserAND)
				}
				{
					p.SetState(104)

					var _x = p.exp(11)

					localctx.(*ExpLogicalAndContext).right = _x
				}

			case 10:
				localctx = NewExpLogicalORContext(p, NewExpContext(p, _parentctx, _parentState))
				localctx.(*ExpLogicalORContext).left = _prevctx

				p.PushNewRecursionContext(localctx, _startState, EqlParserRULE_exp)
				p.SetState(105)

				if !(p.Precpred(p.GetParserRuleContext(), 9)) {
					panic(antlr.NewFailedPredicateException(p, "p.Precpred(p.GetParserRuleContext(), 9)", ""))
				}
				{
					p.SetState(106)
					p.Match(EqlParserOR)
				}
				{
					p.SetState(107)

					var _x = p.exp(10)

					localctx.(*ExpLogicalORContext).right = _x
				}

			}

		}
		p.SetState(112)
		p.GetErrorHandler().Sync(p)
		_alt = p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 8, p.GetParserRuleContext())
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
	p.RuleIndex = EqlParserRULE_arguments
	return p
}

func (*ArgumentsContext) IsArgumentsContext() {}

func NewArgumentsContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ArgumentsContext {
	var p = new(ArgumentsContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = EqlParserRULE_arguments

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
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterArguments(s)
	}
}

func (s *ArgumentsContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitArguments(s)
	}
}

func (s *ArgumentsContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
		return t.VisitArguments(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *EqlParser) Arguments() (localctx IArgumentsContext) {
	localctx = NewArgumentsContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 12, EqlParserRULE_arguments)
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
		p.SetState(113)
		p.exp(0)
	}
	p.SetState(118)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	for _la == EqlParserT__1 {
		{
			p.SetState(114)
			p.Match(EqlParserT__1)
		}
		{
			p.SetState(115)
			p.exp(0)
		}

		p.SetState(120)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)
	}

	return localctx
}

// IArrayContext is an interface to support dynamic dispatch.
type IArrayContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// IsArrayContext differentiates from other interfaces.
	IsArrayContext()
}

type ArrayContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyArrayContext() *ArrayContext {
	var p = new(ArrayContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = EqlParserRULE_array
	return p
}

func (*ArrayContext) IsArrayContext() {}

func NewArrayContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ArrayContext {
	var p = new(ArrayContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = EqlParserRULE_array

	return p
}

func (s *ArrayContext) GetParser() antlr.Parser { return s.parser }

func (s *ArrayContext) AllConstant() []IConstantContext {
	var ts = s.GetTypedRuleContexts(reflect.TypeOf((*IConstantContext)(nil)).Elem())
	var tst = make([]IConstantContext, len(ts))

	for i, t := range ts {
		if t != nil {
			tst[i] = t.(IConstantContext)
		}
	}

	return tst
}

func (s *ArrayContext) Constant(i int) IConstantContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IConstantContext)(nil)).Elem(), i)

	if t == nil {
		return nil
	}

	return t.(IConstantContext)
}

func (s *ArrayContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ArrayContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ArrayContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterArray(s)
	}
}

func (s *ArrayContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitArray(s)
	}
}

func (s *ArrayContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
		return t.VisitArray(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *EqlParser) Array() (localctx IArrayContext) {
	localctx = NewArrayContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 14, EqlParserRULE_array)
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
		p.SetState(121)
		p.Constant()
	}
	p.SetState(126)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	for _la == EqlParserT__1 {
		{
			p.SetState(122)
			p.Match(EqlParserT__1)
		}
		{
			p.SetState(123)
			p.Constant()
		}

		p.SetState(128)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)
	}

	return localctx
}

// IKeyContext is an interface to support dynamic dispatch.
type IKeyContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// IsKeyContext differentiates from other interfaces.
	IsKeyContext()
}

type KeyContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyKeyContext() *KeyContext {
	var p = new(KeyContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = EqlParserRULE_key
	return p
}

func (*KeyContext) IsKeyContext() {}

func NewKeyContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *KeyContext {
	var p = new(KeyContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = EqlParserRULE_key

	return p
}

func (s *KeyContext) GetParser() antlr.Parser { return s.parser }

func (s *KeyContext) Constant() IConstantContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IConstantContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(IConstantContext)
}

func (s *KeyContext) NAME() antlr.TerminalNode {
	return s.GetToken(EqlParserNAME, 0)
}

func (s *KeyContext) STEXT() antlr.TerminalNode {
	return s.GetToken(EqlParserSTEXT, 0)
}

func (s *KeyContext) DTEXT() antlr.TerminalNode {
	return s.GetToken(EqlParserDTEXT, 0)
}

func (s *KeyContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *KeyContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *KeyContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterKey(s)
	}
}

func (s *KeyContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitKey(s)
	}
}

func (s *KeyContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
		return t.VisitKey(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *EqlParser) Key() (localctx IKeyContext) {
	localctx = NewKeyContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 16, EqlParserRULE_key)
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
		p.SetState(129)
		_la = p.GetTokenStream().LA(1)

		if !(((_la)&-(0x1f+1)) == 0 && ((1<<uint(_la))&((1<<EqlParserNAME)|(1<<EqlParserSTEXT)|(1<<EqlParserDTEXT))) != 0) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}
	{
		p.SetState(130)
		p.Match(EqlParserT__2)
	}
	{
		p.SetState(131)
		p.Constant()
	}

	return localctx
}

// IDictContext is an interface to support dynamic dispatch.
type IDictContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// IsDictContext differentiates from other interfaces.
	IsDictContext()
}

type DictContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyDictContext() *DictContext {
	var p = new(DictContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = EqlParserRULE_dict
	return p
}

func (*DictContext) IsDictContext() {}

func NewDictContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *DictContext {
	var p = new(DictContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = EqlParserRULE_dict

	return p
}

func (s *DictContext) GetParser() antlr.Parser { return s.parser }

func (s *DictContext) AllKey() []IKeyContext {
	var ts = s.GetTypedRuleContexts(reflect.TypeOf((*IKeyContext)(nil)).Elem())
	var tst = make([]IKeyContext, len(ts))

	for i, t := range ts {
		if t != nil {
			tst[i] = t.(IKeyContext)
		}
	}

	return tst
}

func (s *DictContext) Key(i int) IKeyContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IKeyContext)(nil)).Elem(), i)

	if t == nil {
		return nil
	}

	return t.(IKeyContext)
}

func (s *DictContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *DictContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *DictContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.EnterDict(s)
	}
}

func (s *DictContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(EqlListener); ok {
		listenerT.ExitDict(s)
	}
}

func (s *DictContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case EqlVisitor:
		return t.VisitDict(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *EqlParser) Dict() (localctx IDictContext) {
	localctx = NewDictContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 18, EqlParserRULE_dict)
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
		p.SetState(133)
		p.Key()
	}
	p.SetState(138)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	for _la == EqlParserT__1 {
		{
			p.SetState(134)
			p.Match(EqlParserT__1)
		}
		{
			p.SetState(135)
			p.Key()
		}

		p.SetState(140)
		p.GetErrorHandler().Sync(p)
		_la = p.GetTokenStream().LA(1)
	}

	return localctx
}

func (p *EqlParser) Sempred(localctx antlr.RuleContext, ruleIndex, predIndex int) bool {
	switch ruleIndex {
	case 5:
		var t *ExpContext = nil
		if localctx != nil {
			t = localctx.(*ExpContext)
		}
		return p.Exp_Sempred(t, predIndex)

	default:
		panic("No predicate with index: " + fmt.Sprint(ruleIndex))
	}
}

func (p *EqlParser) Exp_Sempred(localctx antlr.RuleContext, predIndex int) bool {
	switch predIndex {
	case 0:
		return p.Precpred(p.GetParserRuleContext(), 19)

	case 1:
		return p.Precpred(p.GetParserRuleContext(), 18)

	case 2:
		return p.Precpred(p.GetParserRuleContext(), 16)

	case 3:
		return p.Precpred(p.GetParserRuleContext(), 15)

	case 4:
		return p.Precpred(p.GetParserRuleContext(), 14)

	case 5:
		return p.Precpred(p.GetParserRuleContext(), 13)

	case 6:
		return p.Precpred(p.GetParserRuleContext(), 12)

	case 7:
		return p.Precpred(p.GetParserRuleContext(), 11)

	case 8:
		return p.Precpred(p.GetParserRuleContext(), 10)

	case 9:
		return p.Precpred(p.GetParserRuleContext(), 9)

	default:
		panic("No predicate with index: " + fmt.Sprint(predIndex))
	}
}
