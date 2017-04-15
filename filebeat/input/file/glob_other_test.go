// +build !windows

package file

var globTests = []globTest{
	{
		"*",
		[]string{
			"foo",
		},
	},
	{
		"foo/*",
		[]string{
			"foo/bar",
		},
	},
	{
		"*/*",
		[]string{
			"foo/bar",
		},
	},
	{
		"**",
		[]string{
			"",
			"foo",
			"foo/bar",
			"foo/bar/baz",
			"foo/bar/baz/qux",
		},
	},
	{
		"foo**",
		[]string{
			"foo",
		},
	},
	{
		"foo/**",
		[]string{
			"foo",
			"foo/bar",
			"foo/bar/baz",
			"foo/bar/baz/qux",
			"foo/bar/baz/qux/quux",
		},
	},
	{
		"foo/**/baz",
		[]string{
			"foo/bar/baz",
		},
	},
	{
		"foo/**/bazz",
		[]string{},
	},
	{
		"foo/**/bar",
		[]string{
			"foo/bar",
		},
	},
	{
		"foo//bar",
		[]string{
			"foo/bar",
		},
	},
}

var globPatternsTests = []globPatternsTest{
	{
		"**",
		[]string{"*", "*/*"},
		false,
	},
	{
		"/**",
		[]string{"/", "/*", "/*/*"},
		false,
	},
	{
		"**/",
		[]string{"*", "*/*"},
		false,
	},
	{
		"/foo/**",
		[]string{"/foo", "/foo/*", "/foo/*/*"},
		false,
	},
	{
		"/foo/**/bar",
		[]string{"/foo/bar", "/foo/*/bar", "/foo/*/*/bar"},
		false,
	},
	{
		"**/bar",
		[]string{"bar", "*/bar", "*/*/bar"},
		false,
	},
	{
		"/**/bar",
		[]string{"/bar", "/*/bar", "/*/*/bar"},
		false,
	},
	{
		"**/**",
		[]string{"*", "*/*"},
		true,
	},
	{
		"/**/**",
		[]string{"*", "*/*"},
		true,
	},
	{
		"foo**/bar",
		[]string{"foo**/bar"},
		false,
	},
	{
		"**foo/bar",
		[]string{"**foo/bar"},
		false,
	},
	{
		"foo/**bar",
		[]string{"foo/**bar"},
		false,
	},
	{
		"foo/bar**",
		[]string{"foo/bar**"},
		false,
	},
}
