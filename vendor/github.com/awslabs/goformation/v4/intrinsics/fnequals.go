package intrinsics

import (
	"reflect"
)

// FnEquals resolves the 'Fn::Equals' AWS CloudFormation intrinsic function.
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference-conditions.html#intrinsic-function-reference-conditions-equals
func FnEquals(name string, input interface{}, template interface{}) interface{} {
	// "Fn::Equals" : ["value_1", "value_2"]

	// Check the input is an array
	if arr, ok := input.([]interface{}); ok {
		if len(arr) != 2 {
			return nil
		}

		return reflect.DeepEqual(arr[0], arr[1])
	}

	return nil
}
