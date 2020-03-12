package fmts

func init() {
	const lang = "scala"

	register(&Fmt{
		Name: "scalac",
		Errorformat: []string{
			`%E%f:%l: error: %m`,
			`%W%f:%l: warning: %m`,
			`%A%f:%l: %m`,
			`%Z%p^`,
			`%C%.%#`,
			`%-G%.%#`,
		},
		Description: "Scala compiler",
		URL:         "http://www.scala-lang.org/",
		Language:    lang,
	})

	register(&Fmt{
		Name: "sbt",
		Errorformat: []string{
			`%E[%t%.%+] %f:%l: error: %m`,
			`%A[%t%.%+] %f:%l: %m`,
			`%Z[%.%+] %p^`,
			`%C[%.%+] %.%#`,
			`%-G%.%#`,
		},
		Description: "the interactive build tool",
		URL:         "http://www.scala-sbt.org/",
		Language:    lang,
	})

	register(&Fmt{
		Name: "sbt-scalastyle",
		Errorformat: []string{
			`[%trror] %f:%l:%c: %m`, // [error]
			`[%tarn] %f:%l:%c: %m`,  // [warn]
			`[%trror] %f:%l: %m`,    // [error]
			`[%tarn] %f:%l: %m`,     // [warn]
			`[%trror] %f: %m`,       // [error]
			`[%tarn] %f: %m`,        // [warn]
			`%-G%.%#`,
		},
		Description: "Scalastyle - SBT plugin",
		URL:         "http://www.scalastyle.org/sbt.html",
		Language:    lang,
	})

	register(&Fmt{
		Name: "scalastyle",
		Errorformat: []string{
			`%trror file=%f message=%m line=%l column=%c`,
			`%trror file=%f message=%m line=%l`,
			`%trror file=%f message=%m`,
			`%tarning file=%f message=%m line=%l column=%c`,
			`%tarning file=%f message=%m line=%l`,
			`%tarning file=%f message=%m`,
			`%-G%.%#`,
		},
		Description: "Scalastyle - Command line",
		URL:         "http://www.scalastyle.org/command-line.html",
		Language:    lang,
	})
}
