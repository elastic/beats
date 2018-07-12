package skima

import (
	"time"

	"fmt"
)

var IsDuration = Is("is a duration", func(v interface{}) ValueResult {
	if _, ok := v.(time.Duration); ok {
		return ValidValueResult
	}
	return ValueResult{
		false,
		fmt.Sprintf("Expected a time.duration, got '%v' which is a %T", v, v),
	}
})

func IsEqual(to interface{}) IsDef {
	return Is("equals", func(v interface{}) ValueResult {
		if v == to {
			return ValidValueResult
		}
		return ValueResult{
			false,
			fmt.Sprintf("%v != %v", v, to),
		}
	})
}

var IsNil = Is("is nil", func(v interface{}) ValueResult {
	if v == nil {
		return ValidValueResult
	}
	return ValueResult{
		false,
		fmt.Sprint("Value %v is not nil", v),
	}
})

func intGtChecker(than int) Checker {
	return func(v interface{}) ValueResult {
		n, ok := v.(int)
		if !ok {
			msg := fmt.Sprintf("%v is a %T, but was expecting an int!", v, v)
			return ValueResult{false, msg}
		}

		if n > than {
			return ValidValueResult
		}

		return ValueResult{
			false,
			fmt.Sprintf("%v is not greater than %v", n, than),
		}
	}
}

func IsIntGt(than int) IsDef {
	return Is("greater than", intGtChecker(than))
}
