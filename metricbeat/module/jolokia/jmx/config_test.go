package jmx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanonicalMBeanName(t *testing.T) {
	cases := []struct {
		mbean    string
		expected string
		ok       bool
	}{
		{
			mbean: ``,
			ok:    false,
		},
		{
			mbean: `type=Runtime`,
			ok:    false,
		},
		{
			mbean: `java.lang`,
			ok:    false,
		},
		{
			mbean: `java.lang:`,
			ok:    false,
		},
		{
			mbean: `java.lang:type=Runtime,name`,
			ok:    false,
		},
		{
			mbean:    `java.lang:type=Runtime`,
			expected: `java.lang:type=Runtime`,
			ok:       true,
		},
		{
			mbean:    `java.lang:name=Foo,type=Runtime`,
			expected: `java.lang:name=Foo,type=Runtime`,
			ok:       true,
		},
		{
			mbean:    `java.lang:type=Runtime,name=Foo`,
			expected: `java.lang:name=Foo,type=Runtime`,
			ok:       true,
		},
		{
			mbean:    `java.lang:type=Runtime,name=Foo*`,
			expected: `java.lang:name=Foo*,type=Runtime`,
			ok:       true,
		},
		{
			mbean:    `java.lang:type=Runtime,name=*`,
			expected: `java.lang:name=*,type=Runtime`,
			ok:       true,
		},
		{
			mbean:    `java.lang:type=Runtime,name="foo,bar"`,
			expected: `java.lang:name="foo,bar",type=Runtime`,
			ok:       true,
		},
		{
			mbean:    `Catalina:type=RequestProcessor,worker="http-nio-8080",name=HttpRequest1`,
			expected: `Catalina:name=HttpRequest1,type=RequestProcessor,worker="http-nio-8080"`,
			ok:       true,
		},
	}

	for _, c := range cases {
		canonical, err := canonicalizeMBeanName(c.mbean)
		if c.ok {
			assert.NoError(t, err, "failed parsing for: "+c.mbean)
			assert.Equal(t, c.expected, canonical, "mbean: "+c.mbean)
		} else {
			assert.Error(t, err, "should have failed for: "+c.mbean)
		}
	}
}
