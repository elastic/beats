package fmts

func init() {
	const lang = "env"

	register(&Fmt{
		Name: "dotenv-linter",
		Errorformat: []string{
			`%f:%l %m`,
		},
		Description: "Linter for .env files",
		URL:         "https://github.com/mgrachev/dotenv-linter",
		Language:    lang,
	})
}
