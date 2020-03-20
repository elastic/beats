package intrinsics

var AZs map[string][]interface{} = make(map[string][]interface{})

func buildAZs(region string, zones ...string) (result []interface{}) {
	for _, zone := range zones {
		result = append(result, region+zone)
	}
	return
}

func init() {
	AZs["us-east-1"] = buildAZs("us-east-1", "a", "b", "c", "d")
	AZs["us-west-1"] = buildAZs("us-west-1", "a", "b")
}

// FnGetAZs resolves the 'Fn::GetAZs' AWS CloudFormation intrinsic function.
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference-getavailabilityzones.html
func FnGetAZs(name string, input interface{}, template interface{}) interface{} {

	// Check the input is a string
	if region, ok := input.(string); ok {
		if region == "" {
			region = "us-east-1"
		}

		if azs, ok := AZs[region]; ok {
			return azs
		} else {
			//assume 3 AZs per region
			return buildAZs(region, "a", "b", "c")
		}
	}

	return nil
}
