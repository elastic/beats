package govaluate

/*
  Contains test cases for all the expression examples given in the README.
  While all of the functionality for these cases should be covered in other tests,
  this is really just a sanity check.
*/
import (
	"testing"
)

func TestBasicEvaluation(test *testing.T) {

	expression, err := NewEvaluableExpression("10 > 0")
	if err != nil {
		test.Log(err)
		test.Fail()
	}

	result, err := expression.Evaluate(nil)
	if err != nil {
		test.Log(err)
		test.Fail()
	}

	if result != true {
		test.Logf("Expected 'true', got '%v'\n", result)
		test.Fail()
	}
}

func TestParameterEvaluation(test *testing.T) {

	expression, err := NewEvaluableExpression("foo > 0")
	if err != nil {
		test.Log(err)
		test.Fail()
	}

	parameters := make(map[string]interface{}, 8)
	parameters["foo"] = -1

	result, err := expression.Evaluate(parameters)
	if err != nil {
		test.Log(err)
		test.Fail()
	}

	if result != false {
		test.Logf("Expected 'false', got '%v'\n", result)
		test.Fail()
	}
}

func TestModifierEvaluation(test *testing.T) {

	expression, err := NewEvaluableExpression("(requests_made * requests_succeeded / 100) >= 90")
	if err != nil {
		test.Log(err)
		test.Fail()
	}

	parameters := make(map[string]interface{}, 8)
	parameters["requests_made"] = 100
	parameters["requests_succeeded"] = 80

	result, err := expression.Evaluate(parameters)
	if err != nil {
		test.Log(err)
		test.Fail()
	}

	if result != false {
		test.Logf("Expected 'false', got '%v'\n", result)
		test.Fail()
	}
}

func TestStringEvaluation(test *testing.T) {

	expression, err := NewEvaluableExpression("http_response_body == 'service is ok'")
	if err != nil {
		test.Log(err)
		test.Fail()
	}

	parameters := make(map[string]interface{}, 8)
	parameters["http_response_body"] = "service is ok"

	result, err := expression.Evaluate(parameters)
	if err != nil {
		test.Log(err)
		test.Fail()
	}

	if result != true {
		test.Logf("Expected 'false', got '%v'\n", result)
		test.Fail()
	}
}

func TestFloatEvaluation(test *testing.T) {

	expression, err := NewEvaluableExpression("(mem_used / total_mem) * 100")
	if err != nil {
		test.Log(err)
		test.Fail()
	}

	parameters := make(map[string]interface{}, 8)
	parameters["total_mem"] = 1024
	parameters["mem_used"] = 512

	result, err := expression.Evaluate(parameters)
	if err != nil {
		test.Log(err)
		test.Fail()
	}

	if result != 50.0 {
		test.Logf("Expected '50.0', got '%v'\n", result)
		test.Fail()
	}
}

func TestDateComparison(test *testing.T) {

	expression, err := NewEvaluableExpression("'2014-01-02' > '2014-01-01 23:59:59'")
	if err != nil {
		test.Log(err)
		test.Fail()
	}

	result, err := expression.Evaluate(nil)
	if err != nil {
		test.Log(err)
		test.Fail()
	}

	if result != true {
		test.Logf("Expected 'true', got '%v'\n", result)
		test.Fail()
	}
}

func TestMultipleEvaluation(test *testing.T) {
	expression, _ := NewEvaluableExpression("response_time <= 100")
	parameters := make(map[string]interface{}, 8)

	for i := 0; i < 64; i++ {
		parameters["response_time"] = i
		result, err := expression.Evaluate(parameters)
		if err != nil {
			test.Log(err)
			test.Fail()
		}

		if result != true {
			test.Logf("Expected 'true', got '%v'\n", result)
			test.Fail()
			break
		}
	}
}

func TestStrlenFunction(test *testing.T) {

	functions := map[string]ExpressionFunction{
		"strlen": func(args ...interface{}) (interface{}, error) {
			length := len(args[0].(string))
			return (float64)(length), nil
		},
	}

	expString := "strlen('someReallyLongInputString') <= 16"
	expression, err := NewEvaluableExpressionWithFunctions(expString, functions)
	if err != nil {
		test.Log(err)
		test.Fail()
	}

	result, err := expression.Evaluate(nil)
	if err != nil {
		test.Log(err)
		test.Fail()
	}

	if result != false {
		test.Logf("Expected 'false', got '%v'\n", result)
		test.Fail()
	}
}
