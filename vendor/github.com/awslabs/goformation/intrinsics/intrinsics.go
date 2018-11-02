package intrinsics

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/imdario/mergo"
	yamlwrapper "github.com/sanathkr/yaml"
)

// IntrinsicHandler is a function that applies an intrinsic function and returns
// the response that should be placed in it's place. An intrinsic handler function
// is passed the name of the intrinsic function (e.g. Fn::Join), and the object
// to apply it to (as an interface{}), and should return the resolved object (as an interface{}).
type IntrinsicHandler func(string, interface{}, interface{}) interface{}

// IntrinsicFunctionHandlers is a map of all the possible AWS CloudFormation intrinsic
// functions, and a handler function that is invoked to resolve.
var defaultIntrinsicHandlers = map[string]IntrinsicHandler{
	"Fn::Base64":      FnBase64,
	"Fn::And":         FnAnd,
	"Fn::Equals":      FnEquals,
	"Fn::If":          FnIf,
	"Fn::Not":         FnNot,
	"Fn::Or":          FnOr,
	"Fn::FindInMap":   FnFindInMap,
	"Fn::GetAtt":      nonResolvingHandler,
	"Fn::GetAZs":      FnGetAZs,
	"Fn::ImportValue": nonResolvingHandler,
	"Fn::Join":        FnJoin,
	"Fn::Select":      FnSelect,
	"Fn::Split":       FnSplit,
	"Fn::Sub":         FnSub,
	"Ref":             Ref,
	"Fn::Cidr":        nonResolvingHandler,
}

// ProcessorOptions allows customisation of the intrinsic function processor behaviour.
// This allows disabling the processing of intrinsics,
// overriding of the handlers for each intrinsic function type,
// and overriding template parameters.
type ProcessorOptions struct {
	IntrinsicHandlerOverrides map[string]IntrinsicHandler
	ParameterOverrides        map[string]interface{}
	NoProcess                 bool
}

// nonResolvingHandler is a simple example of an intrinsic function handler function
// that refuses to resolve any intrinsic functions, and just returns a basic string.
func nonResolvingHandler(name string, input interface{}, template interface{}) interface{} {
	return nil
}

// ProcessYAML recursively searches through a byte array of JSON data for all
// AWS CloudFormation intrinsic functions, resolves them, and then returns
// the resulting  interface{} object.
func ProcessYAML(input []byte, options *ProcessorOptions) ([]byte, error) {

	// Convert short form intrinsic functions (e.g. !Sub) to long form
	registerTagMarshallers()

	data, err := yamlwrapper.YAMLToJSON(input)
	if err != nil {
		return nil, fmt.Errorf("invalid YAML template: %s", err)
	}

	return ProcessJSON(data, options)

}

// ProcessJSON recursively searches through a byte array of JSON data for all
// AWS CloudFormation intrinsic functions, resolves them, and then returns
// the resulting  interface{} object.
func ProcessJSON(input []byte, options *ProcessorOptions) ([]byte, error) {

	// First, unmarshal the JSON to a generic interface{} type
	var unmarshalled interface{}
	if err := json.Unmarshal(input, &unmarshalled); err != nil {
		return nil, fmt.Errorf("invalid JSON: %s", err)
	}

	var processed interface{}

	if options != nil && options.NoProcess {
		processed = unmarshalled
	} else {
		applyGlobals(unmarshalled, options)

		overrideParameters(unmarshalled, options)

		evaluateConditions(unmarshalled, options)

		// Process all of the intrinsic functions
		processed = search(unmarshalled, unmarshalled, options)
	}

	// And return the result back as a []byte of JSON
	result, err := json.MarshalIndent(processed, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("invalid JSON: %s", err)
	}

	return result, nil
}

// overrideParameters replaces the default values of Parameters with the specified ones
func overrideParameters(input interface{}, options *ProcessorOptions) {
	if options == nil || len(options.ParameterOverrides) == 0 {
		return
	}

	// Check the template is a map
	if template, ok := input.(map[string]interface{}); ok {
		// Check there is a parameters section
		if uparameters, ok := template["Parameters"]; ok {
			// Check the parameters section is a map
			if parameters, ok := uparameters.(map[string]interface{}); ok {
				for name, value := range options.ParameterOverrides {
					// Check there is a parameter with the same name as the Ref
					if uparameter, ok := parameters[name]; ok {
						// Check the parameter is a map
						if parameter, ok := uparameter.(map[string]interface{}); ok {
							// Set the default value
							parameter["Default"] = value
						}
					}
				}
			}
		}
	}
}

var supportedGlobalResources = map[string]string{
	"Function": "AWS::Serverless::Function",
	"Api":      "AWS::Serverless::Api",
}

// applyGlobals adds AWS SAM Globals into resources
func applyGlobals(input interface{}, options *ProcessorOptions) {
	if template, ok := input.(map[string]interface{}); ok {
		if uglobals, ok := template["Globals"]; ok {
			if globals, ok := uglobals.(map[string]interface{}); ok {
				for name, globalValues := range globals {
					for supportedGlobalName, supportedGlobalType := range supportedGlobalResources {
						if name == supportedGlobalName {
							if uresources, ok := template["Resources"]; ok {
								if resources, ok := uresources.(map[string]interface{}); ok {
									for _, uresource := range resources {
										if resource, ok := uresource.(map[string]interface{}); ok {
											if resource["Type"] == supportedGlobalType {
												properties := resource["Properties"].(map[string]interface{})
												for globalProp, globalPropValue := range globalValues.(map[string]interface{}) {
													if _, ok := properties[globalProp]; !ok {
														properties[globalProp] = globalPropValue
													} else if gArray, ok := globalPropValue.([]interface{}); ok {
														if pArray, ok := properties[globalProp].([]interface{}); ok {
															properties[globalProp] = append(pArray, gArray...)
														}
													} else if gMap, ok := globalPropValue.(map[string]interface{}); ok {
														if pMap, ok := properties[globalProp].(map[string]interface{}); ok {
															mergo.Merge(&pMap, gMap)
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

// evaluateConditions replaces each condition in the template with its corresponding
// value
func evaluateConditions(input interface{}, options *ProcessorOptions) {
	if template, ok := input.(map[string]interface{}); ok {
		// Check there is a conditions section
		if uconditions, ok := template["Conditions"]; ok {
			// Check the conditions section is a map
			if conditions, ok := uconditions.(map[string]interface{}); ok {
				for name, expr := range conditions {
					conditions[name] = search(expr, input, options)
				}
			}
		}
	}
}

// Search is a recursive function, that will search through an interface{} looking for
// an intrinsic function. If it finds one, it calls the provided handler function, passing
// it the type of intrinsic function (e.g. 'Fn::Join'), and the contents. The intrinsic
// handler is expected to return the value that is supposed to be there.
func search(input interface{}, template interface{}, options *ProcessorOptions) interface{} {

	switch value := input.(type) {

	case map[string]interface{}:

		// We've found an object in the JSON, it might be an intrinsic, it might not.
		// To check, we need to see if it contains a specific key that matches the name
		// of an intrinsic function. As golang maps do not guarentee ordering, we need
		// to check every key, not just the first.
		processed := map[string]interface{}{}
		for key, val := range value {

			// See if we have an intrinsic handler function for this object key provided in the
			if h, ok := handler(key, options); ok {
				// This is an intrinsic function, so replace the intrinsic function object
				// with the result of calling the intrinsic function handler for this type
				return h(key, search(val, template, options), template)
			}

			if key == "Condition" {
				// This can lead to infinite recursion A -> B; B -> A;
				// pass state of the conditions that we're evaluating so we can detect cycles
				// in case of cycle, return nil
				return condition(key, search(val, template, options), template, options)
			}

			// This is not an intrinsic function, recurse through it normally
			processed[key] = search(val, template, options)

		}
		return processed

	case []interface{}:

		// We found an array in the JSON - recurse through it's elements looking for intrinsic functions
		processed := []interface{}{}
		for _, val := range value {
			processed = append(processed, search(val, template, options))
		}
		return processed

	case nil:
		return value
	case bool:
		return value
	case float64:
		return value
	case string:

		// Check if the string can be unmarshalled into an intrinsic object
		var decoded []byte
		decoded, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			// The string value is not base64 encoded, so it's not an intrinsic so just pass it back
			return value
		}

		var intrinsic map[string]interface{}
		if err := json.Unmarshal([]byte(decoded), &intrinsic); err != nil {
			// The string value is not JSON, so it's not an intrinsic so just pass it back
			return value
		}

		// An intrinsic should be an object, with a single key containing a valid intrinsic name
		if len(intrinsic) != 1 {
			return value
		}

		for key, val := range intrinsic {
			// See if this is a valid intrinsic function, by comparing the name with our list of registered handlers
			if _, ok := handler(key, options); ok {
				return map[string]interface{}{
					key: search(val, template, options),
				}
			}
		}

		return value
	default:
		return nil

	}

}

// handler looks up the correct intrinsic function handler for an object key, if there is one.
// If not, it returns nil, false.
func handler(name string, options *ProcessorOptions) (IntrinsicHandler, bool) {

	// Check if we have a handler for this intrinsic type in the instrinsic handler
	// overrides in the options provided to Process()
	if options != nil {
		if h, ok := options.IntrinsicHandlerOverrides[name]; ok {
			return h, true
		}
	}

	if h, ok := defaultIntrinsicHandlers[name]; ok {
		return h, true
	}

	return nil, false

}
