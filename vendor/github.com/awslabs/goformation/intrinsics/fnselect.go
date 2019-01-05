package intrinsics

import "strconv"

// FnSelect resolves the 'Fn::Select' AWS CloudFormation intrinsic function.
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference-select.html
func FnSelect(name string, input interface{}, template interface{}) interface{} {

	// { "Fn::Select" : [ index, listOfObjects ] }

	// Check that the input is an array
	if arr, ok := input.([]interface{}); ok {
		// The first element should be the index
		var index int
		if index64, ok := arr[0].(float64); ok {
			index = int(index64)
		} else if indexStr, ok := arr[0].(string); ok {
			if c, err := strconv.Atoi(indexStr); err == nil {
				index = c
			} else {
				return nil
			}
		} else {
			return nil
		}

		// The second element is the array of objects to search
		if objects, ok := arr[1].([]interface{}); ok {
			// Check the requested element is in bounds
			if index < len(objects) {
				return objects[index]
			}
		}
	}

	return nil
}
