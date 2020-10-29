package v2

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestSetFunctions(t *testing.T) {
	cases := []struct {
		name        string
		tfunc       func(ctx transformContext, transformable *transformable, key, val string) error
		paramCtx    transformContext
		paramTr     *transformable
		paramKey    string
		paramVal    string
		expectedTr  *transformable
		expectedErr error
	}{
		{
			name:        "setBody",
			tfunc:       setBody,
			paramCtx:    transformContext{},
			paramTr:     &transformable{body: common.MapStr{}},
			paramKey:    "a_key",
			paramVal:    "a_value",
			expectedTr:  &transformable{body: common.MapStr{"a_key": "a_value"}},
			expectedErr: nil,
		},
		{
			name:        "setHeader",
			tfunc:       setHeader,
			paramCtx:    transformContext{},
			paramTr:     &transformable{header: http.Header{}},
			paramKey:    "a_key",
			paramVal:    "a_value",
			expectedTr:  &transformable{header: http.Header{"A_key": []string{"a_value"}}},
			expectedErr: nil,
		},
		{
			name:     "setURLParams",
			tfunc:    setURLParams,
			paramCtx: transformContext{},
			paramTr: &transformable{url: func() *url.URL {
				u, _ := url.Parse("http://foo.example.com")
				return u
			}()},
			paramKey: "a_key",
			paramVal: "a_value",
			expectedTr: &transformable{url: func() *url.URL {
				u, _ := url.Parse("http://foo.example.com?a_key=a_value")
				return u
			}()},
			expectedErr: nil,
		},
		{
			name:     "setURLValue",
			tfunc:    setURLValue,
			paramCtx: transformContext{},
			paramTr: &transformable{url: func() *url.URL {
				u, _ := url.Parse("http://foo.example.com")
				return u
			}()},
			paramVal: "http://different.example.com",
			expectedTr: &transformable{url: func() *url.URL {
				u, _ := url.Parse("http://different.example.com")
				return u
			}()},
			expectedErr: nil,
		},
	}

	for _, tcase := range cases {
		tcase := tcase
		t.Run(tcase.name, func(t *testing.T) {
			gotErr := tcase.tfunc(tcase.paramCtx, tcase.paramTr, tcase.paramKey, tcase.paramVal)
			if tcase.expectedErr == nil {
				assert.NoError(t, gotErr)
			} else {
				assert.EqualError(t, gotErr, tcase.expectedErr.Error())
			}
			assert.EqualValues(t, tcase.expectedTr, tcase.paramTr)
		})
	}
}
