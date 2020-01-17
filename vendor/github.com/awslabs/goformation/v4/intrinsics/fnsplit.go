package intrinsics

import "strings"

// FnSplit resolves the 'Fn::Split' AWS CloudFormation intrinsic function.
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference-split.html
func FnSplit(name string, input interface{}, template interface{}) interface{} {

	// { "Fn::Split" : [ "delimiter", "source string" ] }

	// Check that the input is an array
	if arr, ok := input.([]interface{}); ok {
		// The first element should be a string (the delimiter)
		if delim, ok := arr[0].(string); ok {
			// The second element should be a string (the content to join)
			if str, ok := arr[1].(string); ok {
				return strings.Split(str, delim)
			}
		}
	}

	return []string{}

}
