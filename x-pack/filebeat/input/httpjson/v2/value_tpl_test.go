package v2

import (
	"net/http"
	"testing"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestValueTpl(t *testing.T) {
	cases := []struct {
		name        string
		value       string
		paramCtx    transformContext
		paramTr     *transformable
		paramDefVal string
		expected    string
	}{
		{
			name:  "canRenderValuesFromCtx",
			value: "{{.last_response.body.param}}",
			paramCtx: transformContext{
				lastResponse: newTransformable(common.MapStr{"param": 25}, nil, ""),
			},
			paramTr:     emptyTransformable(),
			paramDefVal: "",
			expected:    "25",
		},
		{
			name:  "canRenderDefaultValue",
			value: "{{.last_response.body.does_not_exist}}",
			paramCtx: transformContext{
				lastResponse: emptyTransformable(),
			},
			paramTr:     emptyTransformable(),
			paramDefVal: "25",
			expected:    "25",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tpl := &valueTpl{}
			assert.NoError(t, tpl.Unpack(tc.value))
			got := tpl.Execute(tc.paramCtx, tc.paramTr, tc.paramDefVal)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func newTransformable(body common.MapStr, header http.Header, url string) *transformable {
	tr := emptyTransformable()
	if len(body) > 0 {
		tr.body = body
	}
	if len(header) > 0 {
		tr.header = header
	}
	if url != "" {
		tr.url = newURL(url)
	}
	return tr
}
