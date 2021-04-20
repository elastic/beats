package platformcheck

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckPlatformCompat(t *testing.T) {
	if !(runtime.GOARCH == "amd64" && (runtime.GOOS == "linux" ||
		runtime.GOOS == "windows")) {
		t.Skip("Test not support on current platform")
	}

	// compile test helper
	tmp := t.TempDir()
	helper := filepath.Join(tmp, "helper")

	cmd := exec.Command("go", "test", "-c", "-o", helper)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "GOARCH=386")
	require.NoError(t, cmd.Run(), "failed to compile test helper")

	// run test helper
	cmd = exec.Command(helper, "-test.v", "-test.run", "TestHelper")
	cmd.Env = []string{"GO_USE_HELPER=1"}
	output, err := cmd.Output()
	if err != nil {
		t.Logf("32bit binary tester failed.\n Output: %s", output)
	}
}

func TestHelper(t *testing.T) {
	if os.Getenv("GO_USE_HELPER") != "1" {
		t.Log("ignore helper")
		return
	}

	err := CheckNativePlatformCompat()
	if err.Error() != "trying to run 32Bit binary on 64Bit system" {
		t.Error("expected the native platform check to fail")
	}
}
