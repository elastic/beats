package cloudformation

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/awslabs/goformation/v4/intrinsics"
)

func strWrap(fn func(interface{}) string) intrinsics.IntrinsicHandler {
	return func(name string, input interface{}, template interface{}) interface{} {
		return fn(input)
	}
}

func str2Wrap(fn func(interface{}, interface{}) string) intrinsics.IntrinsicHandler {
	return func(name string, input interface{}, template interface{}) interface{} {
		// Check that the input is an array
		if arr, ok := input.([]interface{}); ok {
			if len(arr) != 2 {
				return nil
			}
			return fn(arr[0], arr[1])
		}
		return nil
	}
}

func str3Wrap(fn func(interface{}, interface{}, interface{}) string) intrinsics.IntrinsicHandler {
	return func(name string, input interface{}, template interface{}) interface{} {
		if arr, ok := input.([]interface{}); ok {
			if len(arr) != 3 {
				return nil
			}
			return fn(arr[0], arr[1], arr[2])
		}
		return nil
	}
}

func str2AWrap(fn func(interface{}, []string) string) intrinsics.IntrinsicHandler {
	return func(name string, input interface{}, template interface{}) interface{} {
		if arr, ok := input.([]interface{}); ok {
			switch len(arr) {
			case 0:
				return nil
			case 1:
				return fn(arr[0], []string{})
			case 2:
				if ls, ok := arr[1].([]interface{}); ok {
					return fn(arr[0], interfaceAtostrA(ls))
				}
			}
			return nil
		}
		return nil
	}
}

func strAWrap(fn func([]string) string) intrinsics.IntrinsicHandler {
	return func(name string, input interface{}, template interface{}) interface{} {
		if arr, ok := input.([]interface{}); ok {
			return fn(interfaceAtostrA(arr))
		}
		return nil
	}
}

var EncoderIntrinsics = map[string]intrinsics.IntrinsicHandler{
	"Fn::Base64":      strWrap(Base64),
	"Fn::And":         strAWrap(And),
	"Fn::Equals":      str2Wrap(Equals),
	"Fn::If":          str3Wrap(If),
	"Fn::Not":         strAWrap(Not),
	"Fn::Or":          strAWrap(Or),
	"Fn::FindInMap":   str3Wrap(FindInMap),
	"Fn::GetAtt":      str2Wrap(GetAtt),
	"Fn::GetAZs":      strWrap(GetAZs),
	"Fn::ImportValue": strWrap(ImportValue),
	"Fn::Join":        str2AWrap(Join),
	"Fn::Select":      str2AWrap(Select),
	"Fn::Split":       str2Wrap(Split),
	"Fn::Sub":         strWrap(Sub),
	"Ref":             strWrap(Ref),
	"Fn::Cidr":        str3Wrap(CIDR),
}

// str -> str fns

// Ref creates a CloudFormation Reference to another resource in the template
func Ref(logicalName interface{}) string {
	return encode(fmt.Sprintf(`{ "Ref": "%v" }`, logicalName))
}

// ImportValue returns the value of an output exported by another stack. You typically use this function to create cross-stack references. In the following example template snippets, Stack A exports VPC security group values and Stack B imports them.
func ImportValue(name interface{}) string {
	return encode(fmt.Sprintf(`{ "Fn::ImportValue": "%v" }`, name))
}

// Base64 returns the Base64 representation of the input string. This function is typically used to pass encoded data to Amazon EC2 instances by way of the UserData property
func Base64(input interface{}) string {
	return encode(fmt.Sprintf(`{ "Fn::Base64": "%v" }`, input))
}

// GetAZs returns an array that lists Availability Zones for a specified region. Because customers have access to different Availability Zones, the intrinsic function Fn::GetAZs enables template authors to write templates that adapt to the calling user's access. That way you don't have to hard-code a full list of Availability Zones for a specified region.
func GetAZs(region interface{}) string {
	return encode(fmt.Sprintf(`{ "Fn::GetAZs": "%v" }`, region))
}

// Sub substitutes variables in an input string with values that you specify. In your templates, you can use this function to construct commands or outputs that include values that aren't available until you create or update a stack.
func Sub(value interface{}) string {
	return encode(fmt.Sprintf(`{ "Fn::Sub" : "%v" }`, value))
}

// (str, str) -> str

// GetAtt returns the value of an attribute from a resource in the template.
func GetAtt(logicalName interface{}, attribute interface{}) string {
	return encode(fmt.Sprintf(`{ "Fn::GetAtt" : [ "%v", "%v" ] }`, logicalName, attribute))
}

// Split splits a string into a list of string values so that you can select an element from the resulting string list, use the Fn::Split intrinsic function. Specify the location of splits with a delimiter, such as , (a comma). After you split a string, use the Fn::Select function to pick a specific element.
func Split(delimiter, source interface{}) string {
	return encode(fmt.Sprintf(`{ "Fn::Split" : [ "%v", "%v" ] }`, delimiter, source))
}

// Equals compares if two values are equal. Returns true if the two values are equal or false if they aren't.
func Equals(value1, value2 interface{}) string {
	return encode(fmt.Sprintf(`{ "Fn::Equals" : [ "%v", "%v" ] }`, value1, value2))
}

// (str, str, str) -> str

// CIDR returns an array of CIDR address blocks. The number of CIDR blocks returned is dependent on the count parameter.
func CIDR(ipBlock, count, cidrBits interface{}) string {
	return encode(fmt.Sprintf(`{ "Fn::Cidr" : [ "%v", "%v", "%v" ] }`, ipBlock, count, cidrBits))
}

// FindInMap returns the value corresponding to keys in a two-level map that is declared in the Mappings section.
func FindInMap(mapName, topLevelKey, secondLevelKey interface{}) string {
	return encode(fmt.Sprintf(`{ "Fn::FindInMap" : [ "%v", "%v", "%v" ] }`, mapName, topLevelKey, secondLevelKey))
}

// If returns one value if the specified condition evaluates to true and another value if the specified condition evaluates to false. Currently, AWS CloudFormation supports the Fn::If intrinsic function in the metadata attribute, update policy attribute, and property values in the Resources section and Outputs sections of a template. You can use the AWS::NoValue pseudo parameter as a return value to remove the corresponding property.
func If(value, ifEqual, ifNotEqual interface{}) string {
	return encode(fmt.Sprintf(`{ "Fn::If" : [ "%v", "%v", "%v" ] }`, value, ifEqual, ifNotEqual))
}

// (str, []str) -> str

// Join appends a set of values into a single value, separated by the specified delimiter. If a delimiter is the empty string, the set of values are concatenated with no delimiter.
func Join(delimiter interface{}, values []string) string {
	return encode(fmt.Sprintf(`{ "Fn::Join": [ "%v", [ "%v" ] ] }`, delimiter, strings.Trim(strings.Join(values, `", "`), `, "`)))
}

// Select returns a single object from a list of objects by index.
func Select(index interface{}, list []string) string {
	if len(list) == 1 {
		return encode(fmt.Sprintf(`{ "Fn::Select": [ "%v",  "%v" ] }`, index, list[0]))
	}
	return encode(fmt.Sprintf(`{ "Fn::Select": [ "%v", [ "%v" ] ] }`, index, strings.Trim(strings.Join(list, `", "`), `, "`)))
}

// ([]str) -> str

// And returns true if all the specified conditions evaluate to true, or returns false if any one of the conditions evaluates to false. Fn::And acts as an AND operator. The minimum number of conditions that you can include is 2, and the maximum is 10.
func And(conditions []string) string {
	return encode(fmt.Sprintf(`{ "Fn::And": [ "%v" ] }`, strings.Trim(strings.Join(conditions, `", "`), `, "`)))
}

// Not returns true for a condition that evaluates to false or returns false for a condition that evaluates to true. Fn::Not acts as a NOT operator.
func Not(conditions []string) string {
	return encode(fmt.Sprintf(`{ "Fn::Not": [ "%v" ] }`, strings.Trim(strings.Join(conditions, `", "`), `, "`)))
}

// Or returns true if any one of the specified conditions evaluate to true, or returns false if all of the conditions evaluates to false. Fn::Or acts as an OR operator. The minimum number of conditions that you can include is 2, and the maximum is 10.
func Or(conditions []string) string {
	return encode(fmt.Sprintf(`{ "Fn::Or": [ "%v" ] }`, strings.Trim(strings.Join(conditions, `", "`), `, "`)))
}

// encode takes a string representation of an intrinsic function, and base64 encodes it.
// This prevents the escaping issues when nesting multiple layers of intrinsic functions.
func encode(value string) string {
	return base64.StdEncoding.EncodeToString([]byte(value))
}

func interfaceAtostrA(values []interface{}) []string {
	converted := make([]string, len(values))
	for i := range values {
		converted[i] = fmt.Sprintf("%v", values[i])
	}
	return converted
}
