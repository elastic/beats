package common

import "strconv"

// TryToInt tries to coerce the given interface to an int. On success it returns
// the int value and true.
func TryToInt(number interface{}) (int, bool) {
	var rtn int
	switch v := number.(type) {
	case int:
		rtn = int(v)
	case int8:
		rtn = int(v)
	case int16:
		rtn = int(v)
	case int32:
		rtn = int(v)
	case int64:
		rtn = int(v)
	case uint:
		rtn = int(v)
	case uint8:
		rtn = int(v)
	case uint16:
		rtn = int(v)
	case uint32:
		rtn = int(v)
	case uint64:
		rtn = int(v)
	case string:
		var err error
		rtn, err = strconv.Atoi(v)
		if err != nil {
			return 0, false
		}
	default:
		return 0, false
	}
	return rtn, true
}
