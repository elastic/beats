package generator

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

func DirExists(dir string) bool {
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		return true
	}

	return false
}

func CreateDirectories(baseDir string, directories []string) error {
	for _, d := range directories {
		p := path.Join(baseDir, d)
		err := os.MkdirAll(p, 0750)
		if err != nil {
			return err
		}
	}

	return nil
}

func ReadTemplate(template string, replace map[string]string) ([]byte, error) {
	c, err := ioutil.ReadFile(template)
	if err != nil {
		return []byte{}, fmt.Errorf("cannot read template: %v", err)
	}

	for oldV, newV := range replace {
		c = bytes.Replace(c, []byte("{"+oldV+"}"), []byte(newV), -1)
	}

	return c, nil
}

func CopyTemplates(src, dest string, templates []string, replace map[string]string) error {
	for _, template := range templates {
		err := CopyTemplate(path.Join(src, template), path.Join(dest, template), replace)
		if err != nil {
			return err
		}
	}

	return nil
}

func CopyTemplate(template, dest string, replace map[string]string) error {
	c, err := ReadTemplate(template, replace)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(dest, c, os.ModePerm)
	if err != nil {
		return fmt.Errorf("cannot copy template: %v", err)
	}
	return nil
}

func AppendTemplate(template, dest string, replace map[string]string) error {
	c, err := ReadTemplate(template, replace)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err == nil {
		_, err = f.Write(c)
	}
	if err != nil {
		return fmt.Errorf("cannot append template: %v", err)
	}

	return nil
}
