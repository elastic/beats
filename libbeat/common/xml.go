package common

import (
	"github.com/clbanning/mxj/v2"
)

// UnmarshalXML takes a slice of bytes, and returns a map[string]interface{}.
// If the slice is not valid XML, it will return an error.
func UnmarshalXML(body []byte, prepend bool, toLower bool) (obj map[string]interface{}, err error) {
	var xmlobj mxj.Map
	// Disables attribute prefixes and forces all lines to lowercase to meet ECS standards
	mxj.PrependAttrWithHyphen(prepend)
	mxj.CoerceKeysToLower(toLower)

	xmlobj, err = mxj.NewMapXml(body)
	if err != nil {
		return nil, err
	}

	err = xmlobj.Struct(&obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}
