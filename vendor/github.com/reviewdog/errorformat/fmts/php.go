package fmts

func init() {
	const lang = "php"

	register(&Fmt{
		Name: "phpstan",
		Errorformat: []string{
			`%f:%l:%m`,
		},
		Description: "(phpstan --error-format=raw) PHP Static Analysis Tool - discover bugs in your code without running it!",
		URL:         "https://github.com/phpstan/phpstan",
		Language:    lang,
	})
}
