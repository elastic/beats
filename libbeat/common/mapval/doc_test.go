// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package mapval

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/common"
)

func Example() {
	// Let's say we want to validate this map
	data := common.MapStr{"foo": "bar", "baz": "bot", "count": 1}

	// We can validate the data by creating a mapval.Validator
	// Validators are functions created by compiling the special mapval.Map
	// type. This is a map[string]interface{} that can be compiled
	// into a series of checks.
	//
	// Literal values in a mapval.Map are checked for equality.
	// More complex checks can be done using values of the mapval.IsDef
	// type. In this case, we're using an IsDef to see if the "foo" key
	// contains the string "a", and we're using a literal to check that the
	// "baz" key contains the exact value "bot".
	validator := MustCompile(Map{
		"foo": IsStringContaining("a"),
		"baz": "bot",
	})

	// When being used in test-suites, you should use mapvaltest.Test to execute the validator
	// This produces easy to read test output, and outputs one failed assertion per failed matcher
	// See the docs for mapvaltest for more info
	// mapvaltest.Test(t, validator, data)

	// If you need more control than mapvaltest.Test provides, you can use the results directly
	results := validator(data)

	// The Results.Valid property indicates if the validator passed
	fmt.Printf("Results.Valid: %t\n", results.Valid)

	// Results.Errors() returns one error per failed match
	fmt.Printf("There were %d errors\n", len(results.Errors()))

	// Results.Fields is a map of paths defined in the input mapval.Map to the result of their validation
	// This is useful if you need more control
	fmt.Printf("Over %d fields\n", len(results.Fields))

	// You may be thinking that the validation above should have failed since there was an
	// extra key, 'count', defined that was encountered. By default mapval does not
	// consider extra data to be an error. To change that behavior, wrap the validator
	// in mapval.Strict()
	strictResults := Strict(validator)(data)

	fmt.Printf("Strict Results.Valid: %t\n", strictResults.Valid)

	// You can check an exact field for an error
	fmt.Printf("For the count field specifically .Valid is: %t\n", strictResults.Fields["count"][0].Valid)

	// And get error objects for each error
	for _, err := range strictResults.Errors() {
		fmt.Println(err)
	}

	// And even get a new Results object with only invalid fields included
	strictResults.DetailedErrors()
}

func ExampleCompose() {
	// Composition is useful when you need to share common validation logic between validators.
	// Let's imagine that we want to validate maps describing pets.

	pets := []common.MapStr{
		{"name": "rover", "barks": "often", "fur_length": "long"},
		{"name": "lucky", "barks": "rarely", "fur_length": "short"},
		{"name": "pounce", "meows": "often", "fur_length": "short"},
		{"name": "peanut", "meows": "rarely", "fur_length": "long"},
	}

	// We can see that all pets have the "fur_length" property, but that only cats meow, and dogs bark.
	// We can concisely encode this in mapval using mapval.Compose.
	// We can also see that both "meows" and "barks" contain the same enums of values.
	// We'll start by creating a composed IsDef using the IsAny composition, which creates a new IsDef that is
	// a logical 'or' of its IsDef arguments

	isFrequency := IsAny(IsEqual("often"), IsEqual("rarely"))

	petValidator := MustCompile(Map{
		"name":       IsNonEmptyString,
		"fur_length": IsAny(IsEqual("long"), IsEqual("short")),
	})
	dogValidator := Compose(
		petValidator,
		MustCompile(Map{"barks": isFrequency}),
	)
	catValidator := Compose(
		petValidator,
		MustCompile(Map{"meows": isFrequency}),
	)

	for _, pet := range pets {
		var petType string
		if dogValidator(pet).Valid {
			petType = "dog"
		} else if catValidator(pet).Valid {
			petType = "cat"
		}
		fmt.Printf("%s is a %s\n", pet["name"], petType)
	}

	// Output:
	// rover is a dog
	// lucky is a dog
	// pounce is a cat
	// peanut is a cat
}

func ExampleOptional() {
	dataNoError := common.MapStr{"foo": "bar"}
	dataError := common.MapStr{"foo": "bar", "error": true}

	validator := MustCompile(Map{"foo": "bar", "error": Optional(IsEqual(true))})

	// Both inputs pass
	fmt.Printf("Validator classifies both maps as true: %t", validator(dataNoError).Valid && validator(dataError).Valid)

	// Output:
	// Validator classifies both maps as true: true
}

func ExampleIs() {
	// More advanced validations can be used with built-in and custom functions.
	// These are represented with the IfDef type

	data := common.MapStr{"foo": "bar", "count": 1}

	// Values can also be tested programatically if a mapval.IsDef is used as a value
	// Here we'll define a custom IsDef using the mapval DSL, then validate it.
	// The Is() function is the preferred way to costruct IsDef objects.
	startsWithB := Is("starts with b", func(path path, v interface{}) *Results {
		vStr, ok := v.(string)
		if !ok {
			return SimpleResult(path, false, "Expected a string, got a %t", v)
		}

		if strings.HasPrefix(vStr, "b") {
			return ValidResult(path)
		}

		return SimpleResult(path, false, "Expected string to start with b, got %v", vStr)
	})

	funcValidator := MustCompile(Map{"foo": startsWithB})

	funcValidatorResult := funcValidator(data)

	fmt.Printf("Valid: %t", funcValidatorResult.Valid)

	// Output:
	// Valid: true
}

func ExampleMap() {
	v := MustCompile(Map{
		"foo": IsStringContaining("a"),
		"baz": "bot",
	})

	data := common.MapStr{
		"foo": "bar",
		"baz": "bot",
	}

	fmt.Printf("Result is %t", v(data).Valid)

	// Output:
	// Result is true
}

func ExampleSlice() {
	v := MustCompile(Map{
		"foo": Slice{"foo", IsNonEmptyString},
	})

	data := common.MapStr{"foo": []string{"foo", "something"}}

	fmt.Printf("Result is %t", v(data).Valid)

	// Output:
	// Result is true
}
