package intrinsics

import "encoding/base64"

// FnBase64 resolves the 'Fn::Base64' AWS CloudFormation intrinsic function.
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference-base64.htmlpackage intrinsics
func FnBase64(name string, input interface{}, template interface{}) interface{} {

	// { "Fn::Base64" : valueToEncode }

	// Check the input is a string
	if src, ok := input.(string); ok {
		return base64.StdEncoding.EncodeToString([]byte(src))
	}

	return nil

}
