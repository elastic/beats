package fmts

func init() {
	const lang = "ansible"

	register(&Fmt{
		Name: "ansible-lint",
		Errorformat: []string{
			`%f:%l: %m`,
		},
		Description: "(ansible-lint -p playbook.yml) Checks playbooks for practices and behaviour that could potentially be improved",
		URL:         "https://github.com/ansible/ansible-lint",
		Language:    lang,
	})
}
