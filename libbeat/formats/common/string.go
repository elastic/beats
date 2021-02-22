package common

// ReadString reads a string starting at the given offset
func ReadString(data []byte, offset int) string {
	if offset < 0 || offset >= len(data) {
		return ""
	}

	for end := offset; end < len(data); end++ {
		if data[end] == 0 {
			return string(data[offset:end])
		}
	}
	return ""
}
