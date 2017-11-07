package cfgfile

import (
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGlobWatcher(t *testing.T) {
	// Create random temp directory
	id := strconv.Itoa(rand.New(rand.NewSource(int64(time.Now().Nanosecond()))).Int())
	dir, err := ioutil.TempDir("", id)
	defer os.RemoveAll(dir)
	assert.NoError(t, err)
	glob := dir + "/*.yml"

	gcd := NewGlobWatcher(glob)

	content := []byte("test\n")
	err = ioutil.WriteFile(dir+"/config1.yml", content, 0644)
	assert.NoError(t, err)
	err = ioutil.WriteFile(dir+"/config2.yml", content, 0644)
	assert.NoError(t, err)

	// Make sure not inside compensation time
	time.Sleep(2 * time.Second)

	files, changed, err := gcd.Scan()
	assert.Equal(t, 2, len(files))
	assert.NoError(t, err)
	assert.True(t, changed)

	files, changed, err = gcd.Scan()
	assert.Equal(t, 2, len(files))
	assert.NoError(t, err)
	assert.False(t, changed)

	err = ioutil.WriteFile(dir+"/config3.yml", content, 0644)
	assert.NoError(t, err)

	files, changed, err = gcd.Scan()
	assert.Equal(t, 3, len(files))
	assert.NoError(t, err)
	assert.True(t, changed)

	err = os.Remove(dir + "/config3.yml")
	assert.NoError(t, err)

	files, changed, err = gcd.Scan()
	assert.Equal(t, 2, len(files))
	assert.NoError(t, err)
	assert.True(t, changed)
}
