package fileout

type config struct {
	Index         string `config:"index"`
	Path          string `config:"path"`
	Filename      string `config:"filename"`
	RotateEveryKb int    `config:"rotate_every_kb"`
	NumberOfFiles int    `config:"number_of_files"`
}

var (
	defaultConfig = config{
		NumberOfFiles: 7,
		RotateEveryKb: 10 * 1024,
	}
)
