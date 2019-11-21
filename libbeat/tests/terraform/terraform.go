package terraform

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/stretchr/testify/suite"
)

type TerraformSuite struct {
	suite.Suite

	Dir string

	tmpDir      string
	createdFile string
}

func (s *TerraformSuite) Output(name string) (string, error) {
	return s.createdFile, nil
}

func (s *TerraformSuite) SetupSuite() {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "beats-terraform-")
	s.Require().NoError(err)

	s.T().Logf("TmpDir created: %s", tmpDir)
	s.tmpDir = tmpDir

	s.createdFile = filepath.Join(tmpDir, "file.txt")

	err = ioutil.WriteFile(s.createdFile, []byte("some content"), 0644)
	s.Require().NoError(err)
}

func (s *TerraformSuite) TearDownSuite() {
	err := os.RemoveAll(s.tmpDir)
	s.Require().NoError(err)
}
