// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0

package diag

import (
	"time"
)

// Field to be stored in a context.
type Field struct {
	Key   string
	Value Value

	// Standardized indicates that the field its key and value are standardized
	// according to some external schema. Consumers of a context might decide to
	// handle Standardized and non-standardized fields differently.
	Standardized bool
}

func userField(k string, v Value) Field {
	return Field{Key: k, Value: v}
}

// Bool creates a new user-field storing a bool.
func Bool(key string, b bool) Field { return userField(key, ValBool(b)) }

// Int creates a new user-field storing an int.
func Int(key string, i int) Field { return userField(key, ValInt(i)) }

// Int64 creates a new user-field storing an int64 value.
func Int64(key string, i int64) Field { return userField(key, ValInt64(i)) }

// Uint creates a new user-field storing an uint.
func Uint(key string, i uint) Field { return userField(key, ValUint(i)) }

// Uint64 creates a new user-field storing an uint64.
func Uint64(key string, i uint64) Field { return userField(key, ValUint64(i)) }

// Float creates a new user-field storing a float.
func Float(key string, f float64) Field { return userField(key, ValFloat(f)) }

// String creates a new user-field storing a string.
func String(key, str string) Field { return userField(key, ValString(str)) }

// Duration creates a new user-field storing a duration.
func Duration(key string, dur time.Duration) Field { return userField(key, ValDuration(dur)) }

// Timestamp creates a new user-field storing a time value.
func Timestamp(key string, ts time.Time) Field { return userField(key, ValTime(ts)) }

// Any creates a new user-field storing any value as interface.
func Any(key string, ifc interface{}) Field {
	// TODO: use type switch + reflection to select concrete Field
	return userField(key, ValAny(ifc))
}
