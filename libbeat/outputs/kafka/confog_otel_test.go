package kafka

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractSingleTopic(t *testing.T) {
	testCases := []struct {
		name      string
		tmpl      string
		want      string
		assertErr func(t *testing.T, err error)
	}{
		{
			name:      "Valid template with single attribute",
			tmpl:      "%{[some_field]}",
			want:      "some_field",
			assertErr: nil,
		},
		{
			name:      "Valid template with single attribute with subfield",
			tmpl:      "%{[some_field.subfield]}",
			want:      "some_field.subfield",
			assertErr: nil,
		},
		{
			name:      "Constant template",
			tmpl:      "constant_topic",
			want:      "constant_topic",
			assertErr: nil,
		},
		{
			name: "Template with multiple attributes",
			tmpl: "%{[.field1]}-%{[.field2]}",
			want: "",
			assertErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "only one attribute supported")
			}},
		{
			name: "Template not just an attribute",
			tmpl: "prefix-%{[.field]}",
			want: "",
			assertErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "topic template is more than just a event attribute")
			}},
		{
			name: "Invalid template",
			tmpl: "%{[.invalid_syntax",
			want: "",
			assertErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "failed to compile topic template")
			}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := extractSingleTopic(tc.tmpl)
			if tc.assertErr == nil {
				tc.assertErr = func(t *testing.T, err error) {
					assert.NoError(t, err, "unexpected error")
				}
			}

			tc.assertErr(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
