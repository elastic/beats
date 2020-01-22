package intrinsics

// FnNot resolves the 'Fn::Not' AWS CloudFormation intrinsic function.
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference-conditions.html#intrinsic-function-reference-conditions-not
func FnNot(name string, input interface{}, template interface{}) interface{} {
	// "Fn::Not": [{condition}]

	// Check the input is an array
	if arr, ok := input.([]interface{}); ok {
		if len(arr) != 1 {
			return nil
		}

		if value, ok := retrieveCondition(arr[0], template); ok {
			return !value
		}
	}

	return nil
}
