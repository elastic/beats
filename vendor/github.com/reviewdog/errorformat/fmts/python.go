package fmts

func init() {
	const lang = "python"

	register(&Fmt{
		Name: "pep8",
		Errorformat: []string{
			`%f:%l:%c: %m`,
		},
		Description: "Python style guide checker",
		URL:         "https://pypi.python.org/pypi/pep8",
		Language:    lang,
	})
}
