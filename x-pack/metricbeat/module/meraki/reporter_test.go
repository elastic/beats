package meraki

import "testing"

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{
			name:     "Nil pointer",
			input:    (*int)(nil),
			expected: true,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: true,
		},
		{
			name:     "Non-empty string",
			input:    "test",
			expected: false,
		},
		{
			name:     "Empty slice",
			input:    []string{},
			expected: true,
		},
		{
			name:     "Regular value",
			input:    float64(1.2),
			expected: false,
		},
		{
			name:     "Pointer to int",
			input:    func() *int { i := 42; return &i }(),
			expected: false,
		},
		{
			name:     "Pointer to bool",
			input:    func() *bool { b := false; return &b }(),
			expected: false,
		},
		{
			name:     "Boolean false",
			input:    false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEmpty(tt.input)
			if result != tt.expected {
				t.Errorf("isEmpty(%v) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}
