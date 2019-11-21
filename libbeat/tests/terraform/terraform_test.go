package terraform

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TestTerraformApplySuite struct {
	TerraformSuite

	CreatedFilePath string
}

func (s *TestTerraformApplySuite) TestCheckFileCreated() {
	path, err := s.Output("file_path")
	s.Require().NoError(err)

	s.CreatedFilePath = path

	d, err := ioutil.ReadFile(path)
	s.Require().NoError(err)

	s.Equal("some content", string(d))
}

func TestTerraformApply(t *testing.T) {
	s := new(TestTerraformApplySuite)
	s.Dir = "./test/local"

	suite.Run(t, s)

	// Check that resources were destroyed
	_, err := os.Stat(s.CreatedFilePath)
	assert.True(t, os.IsNotExist(err))
}
