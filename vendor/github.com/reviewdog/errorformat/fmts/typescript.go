package fmts

func init() {
	const lang = "typescript"

	register(&Fmt{
		Name: "tsc",
		Errorformat: []string{
			`%E%f %#(%l,%c): error TS%n: %m`,
			`%E%f %#(%l,%c): error %m`, // fallback
			`%E%f %#(%l,%c): %m`,       // fallback
			`%Eerror %m`,
			`%C%\s%+%m`,
		},
		Description: "TypeScript compiler",
		URL:         "https://www.typescriptlang.org/",
		Language:    lang,
	})

	register(&Fmt{
		Name: "tslint",
		Errorformat: []string{
			`%f[%l, %c]: %m`, // --format=prose
		},
		Description: "An extensible linter for the TypeScript language",
		URL:         "https://github.com/palantir/tslint",
		Language:    lang,
	})

}
