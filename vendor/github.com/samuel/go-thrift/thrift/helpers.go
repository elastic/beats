package thrift

// Bool is a helper routine that allocates a new bool value to store v and returns a pointer to it.
func Bool(v bool) *bool {
	return &v
}

// Float32 is a helper routine that allocates a new float32 value to store v and returns a pointer to it.
func Float32(v float32) *float32 {
	return &v
}

// Float64 is a helper routine that allocates a new float64 value to store v and returns a pointer to it.
func Float64(v float64) *float64 {
	return &v
}

// Byte is a helper routine that allocates a new byte value to store v and returns a pointer to it.
func Byte(v byte) *byte {
	return &v
}

// Int16 is a helper routine that allocates a new int16 value to store v and returns a pointer to it.
func Int16(v int16) *int16 {
	return &v
}

// Int32 is a helper routine that allocates a new int32 value to store v and returns a pointer to it.
func Int32(v int32) *int32 {
	return &v
}

// Int64 is a helper routine that allocates a new int64 value to store v and returns a pointer to it.
func Int64(v int64) *int64 {
	return &v
}

// String is a helper routine that allocates a new string value to store v and returns a pointer to it.
func String(v string) *string {
	return &v
}
