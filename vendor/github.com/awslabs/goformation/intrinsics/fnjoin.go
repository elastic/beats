package intrinsics

import (
	"strings"
)

// FnJoin resolves the 'Fn::Join' AWS CloudFormation intrinsic function.
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference-join.html
func FnJoin(name string, input interface{}, template interface{}) interface{} {

	// Check the input is an array
	if arr, ok := input.([]interface{}); ok {

		switch len(arr) {
		case 0:
			return nil
		case 1:
			return arr[0]
		default:

			// Fn::Join can be used with a delimeter and an array of parts, like so:
			// "Fn::Join": ["," [ "apples", "pears" ]]
			// Or it can be used without a delimiter, and just join the contents
			// "Fn::Join": ["apples", "pears"]
			// Check if the 2nd element of the array is an array, if so, use the first element as the delimiter

			delim := ""
			parts := []string{}
			for i, value := range arr {

				if i == 0 {
					// If the second element is not a string (and is an array), use this first element as a delimiter
					if _, ok := arr[i+1].([]interface{}); ok {
						if d, ok := value.(string); ok {
							delim = d
							continue
						}
					}
				}

				switch v := value.(type) {
				case string:
					// This element is a string; add it to the array of parts that need joining
					parts = append(parts, v)
				case []interface{}:
					// This element is an array; check if it contains strings and add them to the array of parts that need joining
					for _, subvalue := range v {
						if str, ok := subvalue.(string); ok {
							parts = append(parts, str)
						}
					}
				}

			}

			return strings.Join(parts, delim)

		}
	}

	return nil

}
