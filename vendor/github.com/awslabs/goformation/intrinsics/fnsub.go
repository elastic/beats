package intrinsics

import (
	"encoding/base64"
	"regexp"
	"strings"
)

// ResolveFnSub resolves the 'Fn::Sub' AWS CloudFormation intrinsic function.
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference-sub.html
func FnSub(name string, input interface{}, template interface{}) interface{} {

	// Input can either be a string for this type of Fn::Sub call:
	// { "Fn::Sub": "some-string-with-a-${variable}" }

	// or it will be an array of length two for named replacements
	// { "Fn::Sub": [ "some ${replaced}", { "replaced": "value" } ] }

	switch val := input.(type) {

	case []interface{}:
		// Replace each of the variables in element 0 with the items in element 1
		if src, ok := val[0].(string); ok {
			// The seconds element is a map of variables to replace
			if replacements, ok := val[1].(map[string]interface{}); ok {
				// Loop through the replacements
				for key, replacement := range replacements {
					// Check the replacement is a string
					if value, ok := replacement.(string); ok {
						src = strings.Replace(src, "${"+key+"}", value, -1)
					}
				}
				return src
			}
		}

	case string:
		// Look up references for each of the variables
		regex := regexp.MustCompile(`\$\{([\.0-9A-Za-z]+)\}`)
		variables := regex.FindAllStringSubmatch(val, -1)
		for _, variable := range variables {

			var resolved interface{}
			if strings.Contains(variable[1], ".") {
				// If the variable name has a . in it, use Fn::GetAtt to resolve it
				resolved = FnGetAtt("Fn::GetAtt", strings.Split(variable[1], "."), template)
			} else {
				// The variable name doesn't have a . in it, so use Ref
				resolved = Ref("Ref", variable[1], template)
			}

			if resolved != nil {
				if replacement, ok := resolved.(string); ok {
					val = strings.Replace(val, variable[0], replacement, -1)
				}
			} else {
				// The reference couldn't be resolved, so just strip the variable
				val = strings.Replace(val, variable[0], "", -1)
			}

		}
		return val
	}

	return nil

}

// NewSub substitutes variables in an input string with values that you specify. In your templates, you can use this function to construct commands or outputs that include values that aren't available until you create or update a stack.
func Sub(value string) string {
	i := `{ "Fn::Sub" : "` + value + `" }`
	return base64.StdEncoding.EncodeToString([]byte(i))
}
