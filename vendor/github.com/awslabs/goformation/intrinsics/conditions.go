package intrinsics

// condition evaluates a condition
func condition(name string, input interface{}, template interface{}, options *ProcessorOptions) interface{} {
	if v, ok := input.(string); ok {

		if v, ok := retrieveCondition(input, template); ok {
			return v
		}

		if c := getCondition(v, template); c != nil {
			res := search(c, template, options)
			// replace the value in the template so the value can be reused
			setCondition(v, res, template)

			return res
		}
	}

	return nil
}

func setCondition(name string, val interface{}, template interface{}) {
	if template, ok := template.(map[string]interface{}); ok {
		// Check there is a conditions section
		if uconditions, ok := template["Conditions"]; ok {
			// Check the conditions section is a map
			if conditions, ok := uconditions.(map[string]interface{}); ok {
				// Check there is a condition with the same name as the condition
				if _, ok := conditions[name]; ok {
					conditions[name] = val
				}
			}
		}
	}
}

func getCondition(name string, template interface{}) interface{} {
	if template, ok := template.(map[string]interface{}); ok {
		// Check there is a conditions section
		if uconditions, ok := template["Conditions"]; ok {
			// Check the conditions section is a map
			if conditions, ok := uconditions.(map[string]interface{}); ok {
				// Check there is a condition with the same name as the condition
				if ucondition, ok := conditions[name]; ok {
					return ucondition
				}
			}
		}
	}

	return nil
}

func retrieveCondition(cName interface{}, template interface{}) (value bool, found bool) {

	switch v := cName.(type) {
	case string:
		value, found = getCondition(v, template).(bool)
	case bool:
		value, found = v, true
	}

	return
}
