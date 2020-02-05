package intrinsics

// FnIf resolves the 'Fn::If' AWS CloudFormation intrinsic function.
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference-conditions.html#intrinsic-function-reference-conditions-if
func FnIf(name string, input interface{}, template interface{}) interface{} {

	// "Fn::If": [condition_name, value_if_true, value_if_false]

	// Check the input is an array
	if arr, ok := input.([]interface{}); ok {
		if len(arr) != 3 {
			return nil
		}

		if value, ok := retrieveCondition(arr[0], template); ok {
			if value {
				return arr[1]
			} else {
				return arr[2]
			}
		}
	}

	return nil
}
