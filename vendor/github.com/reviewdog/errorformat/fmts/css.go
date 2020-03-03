package fmts

func init() {
	const lang = "css"

	register(&Fmt{
		Name: "stylelint",
		Errorformat: []string{
			`%-P%f`,
			`%*[\ ]%l:%c%*[\ ]%*[✖⚠]%*[\ ]%m`,
			`%-Q`,
		},
		Description: "A mighty modern CSS linter",
		URL:         "https://github.com/stylelint/stylelint",
		Language:    lang,
	})
}
