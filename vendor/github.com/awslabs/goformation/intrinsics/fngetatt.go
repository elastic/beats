package intrinsics

// FnGetAtt is not implemented, and always returns nil.
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference-getatt.html
func FnGetAtt(name string, input interface{}, template interface{}) interface{} {

	// { "Fn::GetAtt" : [ "logicalNameOfResource", "attributeName" ] }
	return nil
}
