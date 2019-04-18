package govaluate

import (
	"fmt"
	"strings"
	"testing"
)

const (
	EXPERR_NIL_VALUE string = "cannot have a nil value"
)

/*
	Contains a single test case for the EvaluableExpression.NewEvaluableExpressionFromTokens() method.

	These tests, and the ones in `tokenExpressionFailure_test` will be fairly incomplete.
	Creating an expression from a string and from tokens _must_ both perform the same syntax checks.
	So all the checks in `parsing_test` will follow the same logic as the ones here.

	These tests check some corner cases - such as tokens having nil values when they must have something.
	Cases that cannot occur through the normal parser, but may occur in other parsers.
*/
type ExpressionTokenSyntaxTest struct {
	Name     string
	Input    []ExpressionToken
	Expected string
}

func TestNilValues(test *testing.T) {

	cases := []ExpressionTokenSyntaxTest{
		ExpressionTokenSyntaxTest{
			Name: "Nil numeric",
			Input: []ExpressionToken{
				ExpressionToken{
					Kind: NUMERIC,
				},
			},
			Expected: EXPERR_NIL_VALUE,
		},
		ExpressionTokenSyntaxTest{
			Name: "Nil string",
			Input: []ExpressionToken{
				ExpressionToken{
					Kind: STRING,
				},
			},
			Expected: EXPERR_NIL_VALUE,
		},
		ExpressionTokenSyntaxTest{
			Name: "Nil bool",
			Input: []ExpressionToken{
				ExpressionToken{
					Kind: BOOLEAN,
				},
			},
			Expected: EXPERR_NIL_VALUE,
		},
		ExpressionTokenSyntaxTest{
			Name: "Nil time",
			Input: []ExpressionToken{
				ExpressionToken{
					Kind: TIME,
				},
			},
			Expected: EXPERR_NIL_VALUE,
		},
		ExpressionTokenSyntaxTest{
			Name: "Nil pattern",
			Input: []ExpressionToken{
				ExpressionToken{
					Kind: PATTERN,
				},
			},
			Expected: EXPERR_NIL_VALUE,
		},
		ExpressionTokenSyntaxTest{
			Name: "Nil variable",
			Input: []ExpressionToken{
				ExpressionToken{
					Kind: VARIABLE,
				},
			},
			Expected: EXPERR_NIL_VALUE,
		},
		ExpressionTokenSyntaxTest{
			Name: "Nil prefix",
			Input: []ExpressionToken{
				ExpressionToken{
					Kind: PREFIX,
				},
			},
			Expected: EXPERR_NIL_VALUE,
		},
		ExpressionTokenSyntaxTest{
			Name: "Nil comparator",
			Input: []ExpressionToken{
				ExpressionToken{
					Kind:  NUMERIC,
					Value: 1.0,
				},
				ExpressionToken{
					Kind: COMPARATOR,
				},
				ExpressionToken{
					Kind:  NUMERIC,
					Value: 1.0,
				},
			},
			Expected: EXPERR_NIL_VALUE,
		},
		ExpressionTokenSyntaxTest{
			Name: "Nil logicalop",
			Input: []ExpressionToken{
				ExpressionToken{
					Kind:  BOOLEAN,
					Value: true,
				},
				ExpressionToken{
					Kind: LOGICALOP,
				},
				ExpressionToken{
					Kind:  BOOLEAN,
					Value: true,
				},
			},
			Expected: EXPERR_NIL_VALUE,
		},
		ExpressionTokenSyntaxTest{
			Name: "Nil modifer",
			Input: []ExpressionToken{
				ExpressionToken{
					Kind:  NUMERIC,
					Value: 1.0,
				},
				ExpressionToken{
					Kind: MODIFIER,
				},
				ExpressionToken{
					Kind:  NUMERIC,
					Value: 1.0,
				},
			},
			Expected: EXPERR_NIL_VALUE,
		},
		ExpressionTokenSyntaxTest{
			Name: "Nil ternary",
			Input: []ExpressionToken{
				ExpressionToken{
					Kind:  BOOLEAN,
					Value: true,
				},
				ExpressionToken{
					Kind: TERNARY,
				},
				ExpressionToken{
					Kind:  BOOLEAN,
					Value: true,
				},
			},
			Expected: EXPERR_NIL_VALUE,
		},
	}

	runExpressionFromTokenTests(cases, true, test)
}

func runExpressionFromTokenTests(cases []ExpressionTokenSyntaxTest, expectFail bool, test *testing.T) {

	var err error

	fmt.Printf("Running %d expression from expression token tests...\n", len(cases))

	for _, testCase := range cases {

		_, err = NewEvaluableExpressionFromTokens(testCase.Input)

		if err != nil {
			if expectFail {

				if !strings.Contains(err.Error(), testCase.Expected) {

					test.Logf("Test '%s' failed", testCase.Name)
					test.Logf("Got error: '%s', expected '%s'", err.Error(), testCase.Expected)
					test.Fail()
				}
				continue
			}

			test.Logf("Test '%s' failed", testCase.Name)
			test.Logf("Got error: '%s'", err)
			test.Fail()
			continue
		} else {
			if expectFail {

				test.Logf("Test '%s' failed", testCase.Name)
				test.Logf("Expected error, found none\n")
				test.Fail()
				continue
			}
		}
	}
}
