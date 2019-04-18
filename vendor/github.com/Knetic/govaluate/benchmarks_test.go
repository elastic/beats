package govaluate

import (
	"testing"
)

/*
  Serves as a "water test" to give an idea of the general overhead of parsing
*/
func BenchmarkSingleParse(bench *testing.B) {

	for i := 0; i < bench.N; i++ {
		NewEvaluableExpression("1")
	}
}

/*
  The most common use case, a single variable, modified slightly, compared to a constant.
  This is the "expected" use case of govaluate.
*/
func BenchmarkSimpleParse(bench *testing.B) {

	for i := 0; i < bench.N; i++ {
		NewEvaluableExpression("(requests_made * requests_succeeded / 100) >= 90")
	}
}

/*
  Benchmarks all syntax possibilities in one expression.
*/
func BenchmarkFullParse(bench *testing.B) {

	var expression string

	// represents all the major syntax possibilities.
	expression = "2 > 1 &&" +
		"'something' != 'nothing' || " +
		"'2014-01-20' < 'Wed Jul  8 23:07:35 MDT 2015' && " +
		"[escapedVariable name with spaces] <= unescaped\\-variableName &&" +
		"modifierTest + 1000 / 2 > (80 * 100 % 2)"

	for i := 0; i < bench.N; i++ {
		NewEvaluableExpression(expression)
	}
}

/*
  Benchmarks the bare-minimum evaluation time
*/
func BenchmarkEvaluationSingle(bench *testing.B) {

	expression, _ := NewEvaluableExpression("1")

	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		expression.Evaluate(nil)
	}
}

/*
  Benchmarks evaluation times of literals (no variables, no modifiers)
*/
func BenchmarkEvaluationNumericLiteral(bench *testing.B) {

	expression, _ := NewEvaluableExpression("(2) > (1)")

	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		expression.Evaluate(nil)
	}
}

/*
  Benchmarks evaluation times of literals with modifiers
*/
func BenchmarkEvaluationLiteralModifiers(bench *testing.B) {

	expression, _ := NewEvaluableExpression("(2) + (2) == (4)")

	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		expression.Evaluate(nil)
	}
}

func BenchmarkEvaluationParameter(bench *testing.B) {

	expression, _ := NewEvaluableExpression("requests_made")
	parameters := map[string]interface{}{
		"requests_made": 99.0,
	}

	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		expression.Evaluate(parameters)
	}
}

/*
  Benchmarks evaluation times of parameters
*/
func BenchmarkEvaluationParameters(bench *testing.B) {

	expression, _ := NewEvaluableExpression("requests_made > requests_succeeded")
	parameters := map[string]interface{}{
		"requests_made":      99.0,
		"requests_succeeded": 90.0,
	}

	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		expression.Evaluate(parameters)
	}
}

/*
  Benchmarks evaluation times of parameters + literals with modifiers
*/
func BenchmarkEvaluationParametersModifiers(bench *testing.B) {

	expression, _ := NewEvaluableExpression("(requests_made * requests_succeeded / 100) >= 90")
	parameters := map[string]interface{}{
		"requests_made":      99.0,
		"requests_succeeded": 90.0,
	}

	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		expression.Evaluate(parameters)
	}
}

/*
  Benchmarks the ludicrously-unlikely worst-case expression,
  one which uses all features.
  This is largely a canary benchmark to make sure that any syntax additions don't
  unnecessarily bloat the evaluation time.
*/
func BenchmarkComplexExpression(bench *testing.B) {

	var expressionString string

	expressionString = "2 > 1 &&" +
		"'something' != 'nothing' || " +
		"'2014-01-20' < 'Wed Jul  8 23:07:35 MDT 2015' && " +
		"[escapedVariable name with spaces] <= unescaped\\-variableName &&" +
		"modifierTest + 1000 / 2 > (80 * 100 % 2)"

	expression, _ := NewEvaluableExpression(expressionString)
	parameters := map[string]interface{}{
		"escapedVariable name with spaces": 99.0,
		"unescaped\\-variableName":         90.0,
		"modifierTest":                     5.0,
	}

	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		expression.Evaluate(parameters)
	}
}

/*
  Benchmarks uncompiled parameter regex operators, which are the most expensive of the lot.
  Note that regex compilation times are unpredictable and wily things. The regex engine has a lot of edge cases
  and possible performance pitfalls. This test doesn't aim to be comprehensive against all possible regex scenarios,
  it is primarily concerned with tracking how much longer it takes to compile a regex at evaluation-time than during parse-time.
*/
func BenchmarkRegexExpression(bench *testing.B) {

	var expressionString string

	expressionString = "(foo !~ bar) && (foobar =~ oba)"

	expression, _ := NewEvaluableExpression(expressionString)
	parameters := map[string]interface{}{
		"foo": "foo",
		"bar": "bar",
		"baz": "baz",
		"oba": ".*oba.*",
	}

	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		expression.Evaluate(parameters)
	}
}

/*
	Benchmarks pre-compilable regex patterns. Meant to serve as a sanity check that constant strings used as regex patterns
	are actually being precompiled.
	Also demonstrates that (generally) compiling a regex at evaluation-time takes an order of magnitude more time than pre-compiling.
*/
func BenchmarkConstantRegexExpression(bench *testing.B) {

	expressionString := "(foo !~ '[bB]az') && (bar =~ '[bB]ar')"
	expression, _ := NewEvaluableExpression(expressionString)

	parameters := map[string]interface{}{
		"foo": "foo",
		"bar": "bar",
	}

	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		expression.Evaluate(parameters)
	}
}

func BenchmarkAccessors(bench *testing.B) {

	expressionString := "foo.Int"
	expression, _ := NewEvaluableExpression(expressionString)

	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		expression.Evaluate(fooFailureParameters)
	}
}

func BenchmarkAccessorMethod(bench *testing.B) {

	expressionString := "foo.Func()"
	expression, _ := NewEvaluableExpression(expressionString)

	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		expression.Evaluate(fooFailureParameters)
	}
}

func BenchmarkAccessorMethodParams(bench *testing.B) {

	expressionString := "foo.FuncArgStr('bonk')"
	expression, _ := NewEvaluableExpression(expressionString)

	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		expression.Evaluate(fooFailureParameters)
	}
}

func BenchmarkNestedAccessors(bench *testing.B) {

	expressionString := "foo.Nested.Funk"
	expression, _ := NewEvaluableExpression(expressionString)

	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		expression.Evaluate(fooFailureParameters)
	}
}
