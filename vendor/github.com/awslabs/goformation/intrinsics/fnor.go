package intrinsics

// FnOr resolves the 'Fn::Or' AWS CloudFormation intrinsic function.
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference-conditions.html#intrinsic-function-reference-conditions-or
func FnOr(name string, input interface{}, template interface{}) interface{} {
	// "Fn::Or": [{condition}, ...]

	// Check the input is an array
	if arr, ok := input.([]interface{}); ok {
		if len(arr) < 2 || len(arr) > 10 {
			return nil
		}

		for _, c := range arr {
			if value, ok := retrieveCondition(c, template); ok {
				if value {
					return true
				}
			} else {
				return nil
			}
		}

		return false
	}

	return nil
}
