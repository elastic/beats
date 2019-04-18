package govaluate

import (
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"
)

/*
	Represents a test of expression evaluation
*/
type EvaluationTest struct {
	Name       string
	Input      string
	Functions  map[string]ExpressionFunction
	Parameters []EvaluationParameter
	Expected   interface{}
}

type EvaluationParameter struct {
	Name  string
	Value interface{}
}

func TestNoParameterEvaluation(test *testing.T) {

	evaluationTests := []EvaluationTest{

		EvaluationTest{

			Name:     "Single PLUS",
			Input:    "51 + 49",
			Expected: 100.0,
		},
		EvaluationTest{

			Name:     "Single MINUS",
			Input:    "100 - 51",
			Expected: 49.0,
		},
		EvaluationTest{

			Name:     "Single BITWISE AND",
			Input:    "100 & 50",
			Expected: 32.0,
		},
		EvaluationTest{

			Name:     "Single BITWISE OR",
			Input:    "100 | 50",
			Expected: 118.0,
		},
		EvaluationTest{

			Name:     "Single BITWISE XOR",
			Input:    "100 ^ 50",
			Expected: 86.0,
		},
		EvaluationTest{

			Name:     "Single shift left",
			Input:    "2 << 1",
			Expected: 4.0,
		},
		EvaluationTest{

			Name:     "Single shift right",
			Input:    "2 >> 1",
			Expected: 1.0,
		},
		EvaluationTest{

			Name:     "Single BITWISE NOT",
			Input:    "~10",
			Expected: -11.0,
		},
		EvaluationTest{

			Name:     "Single MULTIPLY",
			Input:    "5 * 20",
			Expected: 100.0,
		},
		EvaluationTest{

			Name:     "Single DIVIDE",
			Input:    "100 / 20",
			Expected: 5.0,
		},
		EvaluationTest{

			Name:     "Single even MODULUS",
			Input:    "100 % 2",
			Expected: 0.0,
		},
		EvaluationTest{

			Name:     "Single odd MODULUS",
			Input:    "101 % 2",
			Expected: 1.0,
		},
		EvaluationTest{

			Name:     "Single EXPONENT",
			Input:    "10 ** 2",
			Expected: 100.0,
		},
		EvaluationTest{

			Name:     "Compound PLUS",
			Input:    "20 + 30 + 50",
			Expected: 100.0,
		},
		EvaluationTest{

			Name:     "Compound BITWISE AND",
			Input:    "20 & 30 & 50",
			Expected: 16.0,
		},
		EvaluationTest{

			Name:     "Mutiple operators",
			Input:    "20 * 5 - 49",
			Expected: 51.0,
		},
		EvaluationTest{

			Name:     "Parenthesis usage",
			Input:    "100 - (5 * 10)",
			Expected: 50.0,
		},
		EvaluationTest{

			Name:     "Nested parentheses",
			Input:    "50 + (5 * (15 - 5))",
			Expected: 100.0,
		},
		EvaluationTest{

			Name:     "Nested parentheses with bitwise",
			Input:    "100 ^ (23 * (2 | 5))",
			Expected: 197.0,
		},
		EvaluationTest{

			Name:     "Logical OR operation of two clauses",
			Input:    "(1 == 1) || (true == true)",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Logical AND operation of two clauses",
			Input:    "(1 == 1) && (true == true)",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Implicit boolean",
			Input:    "2 > 1",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Compound boolean",
			Input:    "5 < 10 && 1 < 5",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Evaluated true && false operation (for issue #8)",
			Input:    "1 > 10 && 11 > 10",
			Expected: false,
		},
		EvaluationTest{

			Name:     "Evaluated true && false operation (for issue #8)",
			Input:    "true == true && false == true",
			Expected: false,
		},
		EvaluationTest{

			Name:     "Parenthesis boolean",
			Input:    "10 < 50 && (1 != 2 && 1 > 0)",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Comparison of string constants",
			Input:    "'foo' == 'foo'",
			Expected: true,
		},
		EvaluationTest{

			Name:     "NEQ comparison of string constants",
			Input:    "'foo' != 'bar'",
			Expected: true,
		},
		EvaluationTest{

			Name:     "REQ comparison of string constants",
			Input:    "'foobar' =~ 'oba'",
			Expected: true,
		},
		EvaluationTest{

			Name:     "NREQ comparison of string constants",
			Input:    "'foo' !~ 'bar'",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Multiplicative/additive order",
			Input:    "5 + 10 * 2",
			Expected: 25.0,
		},
		EvaluationTest{

			Name:     "Multiple constant multiplications",
			Input:    "10 * 10 * 10",
			Expected: 1000.0,
		},
		EvaluationTest{

			Name:     "Multiple adds/multiplications",
			Input:    "10 * 10 * 10 + 1 * 10 * 10",
			Expected: 1100.0,
		},
		EvaluationTest{

			Name:     "Modulus precedence",
			Input:    "1 + 101 % 2 * 5",
			Expected: 6.0,
		},
		EvaluationTest{

			Name:     "Exponent precedence",
			Input:    "1 + 5 ** 3 % 2 * 5",
			Expected: 6.0,
		},
		EvaluationTest{

			Name:     "Bit shift precedence",
			Input:    "50 << 1 & 90",
			Expected: 64.0,
		},
		EvaluationTest{

			Name:     "Bit shift precedence",
			Input:    "90 & 50 << 1",
			Expected: 64.0,
		},
		EvaluationTest{

			Name:     "Bit shift precedence amongst non-bitwise",
			Input:    "90 + 50 << 1 * 5",
			Expected: 4480.0,
		},
		EvaluationTest{
			Name:     "Order of non-commutative same-precedence operators (additive)",
			Input:    "1 - 2 - 4 - 8",
			Expected: -13.0,
		},
		EvaluationTest{
			Name:     "Order of non-commutative same-precedence operators (multiplicative)",
			Input:    "1 * 4 / 2 * 8",
			Expected: 16.0,
		},
		EvaluationTest{
			Name:     "Null coalesce precedence",
			Input:    "true ?? true ? 100 + 200 : 400",
			Expected: 300.0,
		},
		EvaluationTest{

			Name:     "Identical date equivalence",
			Input:    "'2014-01-02 14:12:22' == '2014-01-02 14:12:22'",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Positive date GT",
			Input:    "'2014-01-02 14:12:22' > '2014-01-02 12:12:22'",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Negative date GT",
			Input:    "'2014-01-02 14:12:22' > '2014-01-02 16:12:22'",
			Expected: false,
		},
		EvaluationTest{

			Name:     "Positive date GTE",
			Input:    "'2014-01-02 14:12:22' >= '2014-01-02 12:12:22'",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Negative date GTE",
			Input:    "'2014-01-02 14:12:22' >= '2014-01-02 16:12:22'",
			Expected: false,
		},
		EvaluationTest{

			Name:     "Positive date LT",
			Input:    "'2014-01-02 14:12:22' < '2014-01-02 16:12:22'",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Negative date LT",
			Input:    "'2014-01-02 14:12:22' < '2014-01-02 11:12:22'",
			Expected: false,
		},
		EvaluationTest{

			Name:     "Positive date LTE",
			Input:    "'2014-01-02 09:12:22' <= '2014-01-02 12:12:22'",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Negative date LTE",
			Input:    "'2014-01-02 14:12:22' <= '2014-01-02 11:12:22'",
			Expected: false,
		},
		EvaluationTest{

			Name:     "Sign prefix comparison",
			Input:    "-1 < 0",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Lexicographic LT",
			Input:    "'ab' < 'abc'",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Lexicographic LTE",
			Input:    "'ab' <= 'abc'",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Lexicographic GT",
			Input:    "'aba' > 'abc'",
			Expected: false,
		},
		EvaluationTest{

			Name:     "Lexicographic GTE",
			Input:    "'aba' >= 'abc'",
			Expected: false,
		},
		EvaluationTest{

			Name:     "Boolean sign prefix comparison",
			Input:    "!true == false",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Inversion of clause",
			Input:    "!(10 < 0)",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Negation after modifier",
			Input:    "10 * -10",
			Expected: -100.0,
		},
		EvaluationTest{

			Name:     "Ternary with single boolean",
			Input:    "true ? 10",
			Expected: 10.0,
		},
		EvaluationTest{

			Name:     "Ternary nil with single boolean",
			Input:    "false ? 10",
			Expected: nil,
		},
		EvaluationTest{

			Name:     "Ternary with comparator boolean",
			Input:    "10 > 5 ? 35.50",
			Expected: 35.50,
		},
		EvaluationTest{

			Name:     "Ternary nil with comparator boolean",
			Input:    "1 > 5 ? 35.50",
			Expected: nil,
		},
		EvaluationTest{

			Name:     "Ternary with parentheses",
			Input:    "(5 * (15 - 5)) > 5 ? 35.50",
			Expected: 35.50,
		},
		EvaluationTest{

			Name:     "Ternary precedence",
			Input:    "true ? 35.50 > 10",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Ternary-else",
			Input:    "false ? 35.50 : 50",
			Expected: 50.0,
		},
		EvaluationTest{

			Name:     "Ternary-else inside clause",
			Input:    "(false ? 5 : 35.50) > 10",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Ternary-else (true-case) inside clause",
			Input:    "(true ? 1 : 5) < 10",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Ternary-else before comparator (negative case)",
			Input:    "true ? 1 : 5 > 10",
			Expected: 1.0,
		},
		EvaluationTest{

			Name:     "Nested ternaries (#32)",
			Input:    "(2 == 2) ? 1 : (true ? 2 : 3)",
			Expected: 1.0,
		},
		EvaluationTest{

			Name:     "Nested ternaries, right case (#32)",
			Input:    "false ? 1 : (true ? 2 : 3)",
			Expected: 2.0,
		},
		EvaluationTest{

			Name:     "Doubly-nested ternaries (#32)",
			Input:    "true ? (false ? 1 : (false ? 2 : 3)) : (false ? 4 : 5)",
			Expected: 3.0,
		},
		EvaluationTest{

			Name:     "String to string concat",
			Input:    "'foo' + 'bar' == 'foobar'",
			Expected: true,
		},
		EvaluationTest{

			Name:     "String to float64 concat",
			Input:    "'foo' + 123 == 'foo123'",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Float64 to string concat",
			Input:    "123 + 'bar' == '123bar'",
			Expected: true,
		},
		EvaluationTest{

			Name:     "String to date concat",
			Input:    "'foo' + '02/05/1970' == 'foobar'",
			Expected: false,
		},
		EvaluationTest{

			Name:     "String to bool concat",
			Input:    "'foo' + true == 'footrue'",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Bool to string concat",
			Input:    "true + 'bar' == 'truebar'",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Null coalesce left",
			Input:    "1 ?? 2",
			Expected: 1.0,
		},
		EvaluationTest{

			Name:     "Array membership literals",
			Input:    "1 in (1, 2, 3)",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Array membership literal with inversion",
			Input:    "!(1 in (1, 2, 3))",
			Expected: false,
		},
		EvaluationTest{

			Name:     "Logical operator reordering (#30)",
			Input:    "(true && true) || (true && false)",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Logical operator reordering without parens (#30)",
			Input:    "true && true || true && false",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Logical operator reordering with multiple OR (#30)",
			Input:    "false || true && true || false",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Left-side multiple consecutive (should be reordered) operators",
			Input:    "(10 * 10 * 10) > 10",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Three-part non-paren logical op reordering (#44)",
			Input:    "false && true || true",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Three-part non-paren logical op reordering (#44), second one",
			Input:    "true || false && true",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Logical operator reordering without parens (#45)",
			Input:    "true && true || false && false",
			Expected: true,
		},
		EvaluationTest{

			Name:  "Single function",
			Input: "foo()",
			Functions: map[string]ExpressionFunction{
				"foo": func(arguments ...interface{}) (interface{}, error) {
					return true, nil
				},
			},

			Expected: true,
		},
		EvaluationTest{

			Name:  "Function with argument",
			Input: "passthrough(1)",
			Functions: map[string]ExpressionFunction{
				"passthrough": func(arguments ...interface{}) (interface{}, error) {
					return arguments[0], nil
				},
			},

			Expected: 1.0,
		},

		EvaluationTest{

			Name:  "Function with arguments",
			Input: "passthrough(1, 2)",
			Functions: map[string]ExpressionFunction{
				"passthrough": func(arguments ...interface{}) (interface{}, error) {
					return arguments[0].(float64) + arguments[1].(float64), nil
				},
			},

			Expected: 3.0,
		},
		EvaluationTest{

			Name:  "Nested function with precedence",
			Input: "sum(1, sum(2, 3), 2 + 2, true ? 4 : 5)",
			Functions: map[string]ExpressionFunction{
				"sum": func(arguments ...interface{}) (interface{}, error) {

					sum := 0.0
					for _, v := range arguments {
						sum += v.(float64)
					}
					return sum, nil
				},
			},

			Expected: 14.0,
		},
		EvaluationTest{

			Name:  "Empty function and modifier, compared",
			Input: "numeric()-1 > 0",
			Functions: map[string]ExpressionFunction{
				"numeric": func(arguments ...interface{}) (interface{}, error) {
					return 2.0, nil
				},
			},

			Expected: true,
		},
		EvaluationTest{

			Name:  "Empty function comparator",
			Input: "numeric() > 0",
			Functions: map[string]ExpressionFunction{
				"numeric": func(arguments ...interface{}) (interface{}, error) {
					return 2.0, nil
				},
			},

			Expected: true,
		},
		EvaluationTest{

			Name:  "Empty function logical operator",
			Input: "success() && !false",
			Functions: map[string]ExpressionFunction{
				"success": func(arguments ...interface{}) (interface{}, error) {
					return true, nil
				},
			},

			Expected: true,
		},
		EvaluationTest{

			Name:  "Empty function ternary",
			Input: "nope() ? 1 : 2.0",
			Functions: map[string]ExpressionFunction{
				"nope": func(arguments ...interface{}) (interface{}, error) {
					return false, nil
				},
			},

			Expected: 2.0,
		},
		EvaluationTest{

			Name:  "Empty function null coalesce",
			Input: "null() ?? 2",
			Functions: map[string]ExpressionFunction{
				"null": func(arguments ...interface{}) (interface{}, error) {
					return nil, nil
				},
			},

			Expected: 2.0,
		},
		EvaluationTest{

			Name:  "Empty function with prefix",
			Input: "-ten()",
			Functions: map[string]ExpressionFunction{
				"ten": func(arguments ...interface{}) (interface{}, error) {
					return 10.0, nil
				},
			},

			Expected: -10.0,
		},
		EvaluationTest{

			Name:  "Empty function as part of chain",
			Input: "10 - numeric() - 2",
			Functions: map[string]ExpressionFunction{
				"numeric": func(arguments ...interface{}) (interface{}, error) {
					return 5.0, nil
				},
			},

			Expected: 3.0,
		},
		EvaluationTest{

			Name:  "Empty function near separator",
			Input: "10 in (1, 2, 3, ten(), 8)",
			Functions: map[string]ExpressionFunction{
				"ten": func(arguments ...interface{}) (interface{}, error) {
					return 10.0, nil
				},
			},

			Expected: true,
		},
		EvaluationTest{

			Name:  "Enclosed empty function with modifier and comparator (#28)",
			Input: "(ten() - 1) > 3",
			Functions: map[string]ExpressionFunction{
				"ten": func(arguments ...interface{}) (interface{}, error) {
					return 10.0, nil
				},
			},

			Expected: true,
		},
		EvaluationTest{
			
			Name:  "Ternary/Java EL ambiguity",
			Input: "false ? foo:length()",
			Functions: map[string]ExpressionFunction{
				"length": func(arguments ...interface{}) (interface{}, error) {
					return 1.0, nil
				},
			},
			Expected: 1.0,
		},
	}

	runEvaluationTests(evaluationTests, test)
}

func TestParameterizedEvaluation(test *testing.T) {

	evaluationTests := []EvaluationTest{

		EvaluationTest{

			Name:  "Single parameter modified by constant",
			Input: "foo + 2",
			Parameters: []EvaluationParameter{

				EvaluationParameter{
					Name:  "foo",
					Value: 2.0,
				},
			},
			Expected: 4.0,
		},
		EvaluationTest{

			Name:  "Single parameter modified by variable",
			Input: "foo * bar",
			Parameters: []EvaluationParameter{

				EvaluationParameter{
					Name:  "foo",
					Value: 5.0,
				},
				EvaluationParameter{
					Name:  "bar",
					Value: 2.0,
				},
			},
			Expected: 10.0,
		},
		EvaluationTest{

			Name:  "Multiple multiplications of the same parameter",
			Input: "foo * foo * foo",
			Parameters: []EvaluationParameter{

				EvaluationParameter{
					Name:  "foo",
					Value: 10.0,
				},
			},
			Expected: 1000.0,
		},
		EvaluationTest{

			Name:  "Multiple additions of the same parameter",
			Input: "foo + foo + foo",
			Parameters: []EvaluationParameter{

				EvaluationParameter{
					Name:  "foo",
					Value: 10.0,
				},
			},
			Expected: 30.0,
		},
		EvaluationTest{

			Name:  "Parameter name sensitivity",
			Input: "foo + FoO + FOO",
			Parameters: []EvaluationParameter{

				EvaluationParameter{
					Name:  "foo",
					Value: 8.0,
				},
				EvaluationParameter{
					Name:  "FoO",
					Value: 4.0,
				},
				EvaluationParameter{
					Name:  "FOO",
					Value: 2.0,
				},
			},
			Expected: 14.0,
		},
		EvaluationTest{

			Name:  "Sign prefix comparison against prefixed variable",
			Input: "-1 < -foo",
			Parameters: []EvaluationParameter{

				EvaluationParameter{
					Name:  "foo",
					Value: -8.0,
				},
			},
			Expected: true,
		},
		EvaluationTest{

			Name:  "Fixed-point parameter",
			Input: "foo > 1",
			Parameters: []EvaluationParameter{

				EvaluationParameter{
					Name:  "foo",
					Value: 2,
				},
			},
			Expected: true,
		},
		EvaluationTest{

			Name:     "Modifier after closing clause",
			Input:    "(2 + 2) + 2 == 6",
			Expected: true,
		},
		EvaluationTest{

			Name:     "Comparator after closing clause",
			Input:    "(2 + 2) >= 4",
			Expected: true,
		},
		EvaluationTest{

			Name:  "Two-boolean logical operation (for issue #8)",
			Input: "(foo == true) || (bar == true)",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "foo",
					Value: true,
				},
				EvaluationParameter{
					Name:  "bar",
					Value: false,
				},
			},
			Expected: true,
		},
		EvaluationTest{

			Name:  "Two-variable integer logical operation (for issue #8)",
			Input: "foo > 10 && bar > 10",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "foo",
					Value: 1,
				},
				EvaluationParameter{
					Name:  "bar",
					Value: 11,
				},
			},
			Expected: false,
		},
		EvaluationTest{

			Name:  "Regex against right-hand parameter",
			Input: "'foobar' =~ foo",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "foo",
					Value: "obar",
				},
			},
			Expected: true,
		},
		EvaluationTest{

			Name:  "Not-regex against right-hand parameter",
			Input: "'foobar' !~ foo",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "foo",
					Value: "baz",
				},
			},
			Expected: true,
		},
		EvaluationTest{

			Name:  "Regex against two parameters",
			Input: "foo =~ bar",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "foo",
					Value: "foobar",
				},
				EvaluationParameter{
					Name:  "bar",
					Value: "oba",
				},
			},
			Expected: true,
		},
		EvaluationTest{

			Name:  "Not-regex against two parameters",
			Input: "foo !~ bar",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "foo",
					Value: "foobar",
				},
				EvaluationParameter{
					Name:  "bar",
					Value: "baz",
				},
			},
			Expected: true,
		},
		EvaluationTest{

			Name:  "Pre-compiled regex",
			Input: "foo =~ bar",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "foo",
					Value: "foobar",
				},
				EvaluationParameter{
					Name:  "bar",
					Value: regexp.MustCompile("[fF][oO]+"),
				},
			},
			Expected: true,
		},
		EvaluationTest{

			Name:  "Pre-compiled not-regex",
			Input: "foo !~ bar",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "foo",
					Value: "foobar",
				},
				EvaluationParameter{
					Name:  "bar",
					Value: regexp.MustCompile("[fF][oO]+"),
				},
			},
			Expected: false,
		},
		EvaluationTest{

			Name:  "Single boolean parameter",
			Input: "commission ? 10",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "commission",
					Value: true,
				},
			},
			Expected: 10.0,
		},
		EvaluationTest{

			Name:  "True comparator with a parameter",
			Input: "partner == 'amazon' ? 10",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "partner",
					Value: "amazon",
				},
			},
			Expected: 10.0,
		},
		EvaluationTest{

			Name:  "False comparator with a parameter",
			Input: "partner == 'amazon' ? 10",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "partner",
					Value: "ebay",
				},
			},
			Expected: nil,
		},
		EvaluationTest{

			Name:  "True comparator with multiple parameters",
			Input: "theft && period == 24 ? 60",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "theft",
					Value: true,
				},
				EvaluationParameter{
					Name:  "period",
					Value: 24,
				},
			},
			Expected: 60.0,
		},
		EvaluationTest{

			Name:  "False comparator with multiple parameters",
			Input: "theft && period == 24 ? 60",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "theft",
					Value: false,
				},
				EvaluationParameter{
					Name:  "period",
					Value: 24,
				},
			},
			Expected: nil,
		},
		EvaluationTest{

			Name:  "String concat with single string parameter",
			Input: "foo + 'bar'",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "foo",
					Value: "baz",
				},
			},
			Expected: "bazbar",
		},
		EvaluationTest{

			Name:  "String concat with multiple string parameter",
			Input: "foo + bar",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "foo",
					Value: "baz",
				},
				EvaluationParameter{
					Name:  "bar",
					Value: "quux",
				},
			},
			Expected: "bazquux",
		},
		EvaluationTest{

			Name:  "String concat with float parameter",
			Input: "foo + bar",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "foo",
					Value: "baz",
				},
				EvaluationParameter{
					Name:  "bar",
					Value: 123.0,
				},
			},
			Expected: "baz123",
		},
		EvaluationTest{

			Name:  "Mixed multiple string concat",
			Input: "foo + 123 + 'bar' + true",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "foo",
					Value: "baz",
				},
			},
			Expected: "baz123bartrue",
		},
		EvaluationTest{

			Name:  "Integer width spectrum",
			Input: "uint8 + uint16 + uint32 + uint64 + int8 + int16 + int32 + int64",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "uint8",
					Value: uint8(0),
				},
				EvaluationParameter{
					Name:  "uint16",
					Value: uint16(0),
				},
				EvaluationParameter{
					Name:  "uint32",
					Value: uint32(0),
				},
				EvaluationParameter{
					Name:  "uint64",
					Value: uint64(0),
				},
				EvaluationParameter{
					Name:  "int8",
					Value: int8(0),
				},
				EvaluationParameter{
					Name:  "int16",
					Value: int16(0),
				},
				EvaluationParameter{
					Name:  "int32",
					Value: int32(0),
				},
				EvaluationParameter{
					Name:  "int64",
					Value: int64(0),
				},
			},
			Expected: 0.0,
		},
		EvaluationTest{

			Name:  "Floats",
			Input: "float32 + float64",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "float32",
					Value: float32(0.0),
				},
				EvaluationParameter{
					Name:  "float64",
					Value: float64(0.0),
				},
			},
			Expected: 0.0,
		},
		EvaluationTest{

			Name:  "Null coalesce right",
			Input: "foo ?? 1.0",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "foo",
					Value: nil,
				},
			},
			Expected: 1.0,
		},
		EvaluationTest{

			Name:  "Multiple comparator/logical operators (#30)",
			Input: "(foo >= 2887057408 && foo <= 2887122943) || (foo >= 168100864 && foo <= 168118271)",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "foo",
					Value: 2887057409,
				},
			},
			Expected: true,
		},
		EvaluationTest{

			Name:  "Multiple comparator/logical operators, opposite order (#30)",
			Input: "(foo >= 168100864 && foo <= 168118271) || (foo >= 2887057408 && foo <= 2887122943)",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "foo",
					Value: 2887057409,
				},
			},
			Expected: true,
		},
		EvaluationTest{

			Name:  "Multiple comparator/logical operators, small value (#30)",
			Input: "(foo >= 2887057408 && foo <= 2887122943) || (foo >= 168100864 && foo <= 168118271)",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "foo",
					Value: 168100865,
				},
			},
			Expected: true,
		},
		EvaluationTest{

			Name:  "Multiple comparator/logical operators, small value, opposite order (#30)",
			Input: "(foo >= 168100864 && foo <= 168118271) || (foo >= 2887057408 && foo <= 2887122943)",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "foo",
					Value: 168100865,
				},
			},
			Expected: true,
		},
		EvaluationTest{

			Name:  "Incomparable array equality comparison",
			Input: "arr == arr",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "arr",
					Value: []int{0, 0, 0},
				},
			},
			Expected: true,
		},
		EvaluationTest{

			Name:  "Incomparable array not-equality comparison",
			Input: "arr != arr",
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "arr",
					Value: []int{0, 0, 0},
				},
			},
			Expected: false,
		},
		EvaluationTest{

			Name:  "Mixed function and parameters",
			Input: "sum(1.2, amount) + name",
			Functions: map[string]ExpressionFunction{
				"sum": func(arguments ...interface{}) (interface{}, error) {

					sum := 0.0
					for _, v := range arguments {
						sum += v.(float64)
					}
					return sum, nil
				},
			},
			Parameters: []EvaluationParameter{
				EvaluationParameter{
					Name:  "amount",
					Value: .8,
				},
				EvaluationParameter{
					Name:  "name",
					Value: "awesome",
				},
			},

			Expected: "2awesome",
		},
		EvaluationTest{

			Name:  "Short-circuit OR",
			Input: "true || fail()",
			Functions: map[string]ExpressionFunction{
				"fail": func(arguments ...interface{}) (interface{}, error) {
					return nil, errors.New("Did not short-circuit")
				},
			},
			Expected: true,
		},
		EvaluationTest{

			Name:  "Short-circuit AND",
			Input: "false && fail()",
			Functions: map[string]ExpressionFunction{
				"fail": func(arguments ...interface{}) (interface{}, error) {
					return nil, errors.New("Did not short-circuit")
				},
			},
			Expected: false,
		},
		EvaluationTest{

			Name:  "Short-circuit ternary",
			Input: "true ? 1 : fail()",
			Functions: map[string]ExpressionFunction{
				"fail": func(arguments ...interface{}) (interface{}, error) {
					return nil, errors.New("Did not short-circuit")
				},
			},
			Expected: 1.0,
		},
		EvaluationTest{

			Name:  "Short-circuit coalesce",
			Input: "'foo' ?? fail()",
			Functions: map[string]ExpressionFunction{
				"fail": func(arguments ...interface{}) (interface{}, error) {
					return nil, errors.New("Did not short-circuit")
				},
			},
			Expected: "foo",
		},
		EvaluationTest{

			Name:       "Simple parameter call",
			Input:      "foo.String",
			Parameters: []EvaluationParameter{fooParameter},
			Expected:   fooParameter.Value.(dummyParameter).String,
		},
		EvaluationTest{

			Name:       "Simple parameter function call",
			Input:      "foo.Func()",
			Parameters: []EvaluationParameter{fooParameter},
			Expected:   "funk",
		},
		EvaluationTest{

			Name:       "Simple parameter call from pointer",
			Input:      "fooptr.String",
			Parameters: []EvaluationParameter{fooPtrParameter},
			Expected:   fooParameter.Value.(dummyParameter).String,
		},
		EvaluationTest{

			Name:       "Simple parameter function call from pointer",
			Input:      "fooptr.Func()",
			Parameters: []EvaluationParameter{fooPtrParameter},
			Expected:   "funk",
		},
		EvaluationTest{

			Name:       "Simple parameter function call from pointer",
			Input:      "fooptr.Func3()",
			Parameters: []EvaluationParameter{fooPtrParameter},
			Expected:   "fronk",
		},
		EvaluationTest{

			Name:       "Simple parameter call",
			Input:      "foo.String == 'hi'",
			Parameters: []EvaluationParameter{fooParameter},
			Expected:   false,
		},
		EvaluationTest{

			Name:       "Simple parameter call with modifier",
			Input:      "foo.String + 'hi'",
			Parameters: []EvaluationParameter{fooParameter},
			Expected:   fooParameter.Value.(dummyParameter).String + "hi",
		},
		EvaluationTest{

			Name:       "Simple parameter function call, two-arg return",
			Input:      "foo.Func2()",
			Parameters: []EvaluationParameter{fooParameter},
			Expected:   "frink",
		},
		EvaluationTest{

			Name:       "Parameter function call with all argument types",
			Input:      "foo.TestArgs(\"hello\", 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1.0, 2.0, true)",
			Parameters: []EvaluationParameter{fooParameter},
			Expected:   "hello: 33",
		},

		EvaluationTest{

			Name:       "Simple parameter function call, one arg",
			Input:      "foo.FuncArgStr('boop')",
			Parameters: []EvaluationParameter{fooParameter},
			Expected:   "boop",
		},
		EvaluationTest{

			Name:       "Simple parameter function call, one arg",
			Input:      "foo.FuncArgStr('boop') + 'hi'",
			Parameters: []EvaluationParameter{fooParameter},
			Expected:   "boophi",
		},
		EvaluationTest{

			Name:       "Nested parameter function call",
			Input:      "foo.Nested.Dunk('boop')",
			Parameters: []EvaluationParameter{fooParameter},
			Expected:   "boopdunk",
		},
		EvaluationTest{

			Name:       "Nested parameter call",
			Input:      "foo.Nested.Funk",
			Parameters: []EvaluationParameter{fooParameter},
			Expected:   "funkalicious",
		},
		EvaluationTest{

			Name:       "Parameter call with + modifier",
			Input:      "1 + foo.Int",
			Parameters: []EvaluationParameter{fooParameter},
			Expected:   102.0,
		},
		EvaluationTest{

			Name:       "Parameter string call with + modifier",
			Input:      "'woop' + (foo.String)",
			Parameters: []EvaluationParameter{fooParameter},
			Expected:   "woopstring!",
		},
		EvaluationTest{

			Name:       "Parameter call with && operator",
			Input:      "true && foo.BoolFalse",
			Parameters: []EvaluationParameter{fooParameter},
			Expected:   false,
		},
		EvaluationTest{

			Name:       "Null coalesce nested parameter",
			Input:      "foo.Nil ?? false",
			Parameters: []EvaluationParameter{fooParameter},
			Expected:   false,
		},
	}

	runEvaluationTests(evaluationTests, test)
}

/*
	Tests the behavior of a nil set of parameters.
*/
func TestNilParameters(test *testing.T) {

	expression, _ := NewEvaluableExpression("true")
	_, err := expression.Evaluate(nil)

	if err != nil {
		test.Fail()
	}
}

/*
	Tests functionality related to using functions with a struct method receiver.
	Created to test #54.
*/
func TestStructFunctions(test *testing.T) {

	parseFormat := "2006"
	y2k, _ := time.Parse(parseFormat, "2000")
	y2k1, _ := time.Parse(parseFormat, "2001")

	functions := map[string]ExpressionFunction{
		"func1": func(args ...interface{}) (interface{}, error) {
			return float64(y2k.Year()), nil
		},
		"func2": func(args ...interface{}) (interface{}, error) {
			return float64(y2k1.Year()), nil
		},
	}

	exp, _ := NewEvaluableExpressionWithFunctions("func1() + func2()", functions)
	result, _ := exp.Evaluate(nil)

	if result != 4001.0 {
		test.Logf("Function calling method did not return the right value. Got: %v, expected %d\n", result, 4001)
		test.Fail()
	}
}

func runEvaluationTests(evaluationTests []EvaluationTest, test *testing.T) {

	var expression *EvaluableExpression
	var result interface{}
	var parameters map[string]interface{}
	var err error

	fmt.Printf("Running %d evaluation test cases...\n", len(evaluationTests))

	// Run the test cases.
	for _, evaluationTest := range evaluationTests {

		if evaluationTest.Functions != nil {
			expression, err = NewEvaluableExpressionWithFunctions(evaluationTest.Input, evaluationTest.Functions)
		} else {
			expression, err = NewEvaluableExpression(evaluationTest.Input)
		}

		if err != nil {

			test.Logf("Test '%s' failed to parse: '%s'", evaluationTest.Name, err)
			test.Fail()
			continue
		}

		parameters = make(map[string]interface{}, 8)

		for _, parameter := range evaluationTest.Parameters {
			parameters[parameter.Name] = parameter.Value
		}

		result, err = expression.Evaluate(parameters)

		if err != nil {

			test.Logf("Test '%s' failed", evaluationTest.Name)
			test.Logf("Encountered error: %s", err.Error())
			test.Fail()
			continue
		}

		if result != evaluationTest.Expected {

			test.Logf("Test '%s' failed", evaluationTest.Name)
			test.Logf("Evaluation result '%v' does not match expected: '%v'", result, evaluationTest.Expected)
			test.Fail()
		}
	}
}
