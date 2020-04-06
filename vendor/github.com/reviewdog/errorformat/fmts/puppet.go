package fmts

func init() {
	const lang = "puppet"

	register(&Fmt{
		Name: "puppet-lint",
		Errorformat: []string{
			`%f - %m on line %l`,
		},
		Description: "Check that your Puppet manifests conform to the style guide",
		URL:         "https://github.com/rodjek/puppet-lint",
		Language:    lang,
	})
}
