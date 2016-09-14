package s3

import (
	"fmt"
)

type config struct {
	Path          string `config:"path"`
	Filename      string `config:"filename"`
	UploadEveryKb int    `config:"upload_every_kb" validate:"min=1"`
	NumberOfFiles int    `config:"number_of_files"`
	Region        string `config:"region"`
	Bucket        string `config:"bucket"`
}

var (
	defaultConfig = config{
		NumberOfFiles: 2,
		UploadEveryKb: 10 * 1024,
		Region:        "us-east-1",
	}
)

func (c *config) Validate() error {
	if c.NumberOfFiles < 2 || c.NumberOfFiles > managerMaxFiles {
		return fmt.Errorf("S3 number_of_files to keep should be between 2 and %v",
			managerMaxFiles)
	}

	return nil
}
