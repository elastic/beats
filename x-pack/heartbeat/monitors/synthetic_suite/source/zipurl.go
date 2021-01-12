package source

type ZipURLSource struct {
	Url string `config:"url"`
	Headers map[string]string `config:"headers"`
	PollingSource
}

func (z *ZipURLSource) Fetch() error {
	panic("implement me")
}

func (z *ZipURLSource) Workdir() string {
	panic("implement me")
}
