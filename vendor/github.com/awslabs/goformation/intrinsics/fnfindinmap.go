package intrinsics

// FnFindInMap resolves the 'Fn::FindInMap' AWS CloudFormation intrinsic function.
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference-findinmap.html
func FnFindInMap(name string, input interface{}, template interface{}) interface{} {

	// { "Fn::FindInMap" : [ "MapName", "TopLevelKey", "SecondLevelKey"] }

	// "Mappings" : {
	// 	"RegionMap" : {
	// 		"us-east-1" : { "32" : "ami-6411e20d", "64" : "ami-7a11e213" },
	// 		"us-west-1" : { "32" : "ami-c9c7978c", "64" : "ami-cfc7978a" },
	// 		"eu-west-1" : { "32" : "ami-37c2f643", "64" : "ami-31c2f645" },
	// 		"ap-southeast-1" : { "32" : "ami-66f28c34", "64" : "ami-60f28c32" },
	// 		"ap-northeast-1" : { "32" : "ami-9c03a89d", "64" : "ami-a003a8a1" }
	// 	}
	// }

	// Holy nesting batman! I'm sure there's a better way to do this... :)

	// Check that the input is an array
	if arr, ok := input.([]interface{}); ok {
		// The first element should be the map name
		if mapname, ok := arr[0].(string); ok {
			// The second element should be the first level map key
			if key1, ok := arr[1].(string); ok {
				// The third element should be the second level map key
				if key2, ok := arr[2].(string); ok {
					// Check the map exists in the CloudFormation template
					if tmpl, ok := template.(map[string]interface{}); ok {
						if mappings, ok := tmpl["Mappings"]; ok {
							if mapmap, ok := mappings.(map[string]interface{}); ok {
								if found, ok := mapmap[mapname]; ok {
									if foundmap, ok := found.(map[string]interface{}); ok {
										// Ok, we've got the map, check the first key exists
										if foundkey1, ok := foundmap[key1]; ok {
											if foundkey1map, ok := foundkey1.(map[string]interface{}); ok {
												if foundkey2, ok := foundkey1map[key2]; ok {
													return foundkey2
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

	return nil

}
