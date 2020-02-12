package mg

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

// CacheEnv is the environment variable that users may set to change the
// location where mage stores its compiled binaries.
const CacheEnv = "MAGEFILE_CACHE"

// VerboseEnv is the environment variable that indicates the user requested
// verbose mode when running a magefile.
const VerboseEnv = "MAGEFILE_VERBOSE"

// DebugEnv is the environment variable that indicates the user requested
// debug mode when running mage.
const DebugEnv = "MAGEFILE_DEBUG"

// GoCmdEnv is the environment variable that indicates the go binary the user
// desires to utilize for Magefile compilation.
const GoCmdEnv = "MAGEFILE_GOCMD"

// IgnoreDefaultEnv is the environment variable that indicates the user requested
// to ignore the default target specified in the magefile.
const IgnoreDefaultEnv = "MAGEFILE_IGNOREDEFAULT"

// HashFastEnv is the environment variable that indicates the user requested to
// use a quick hash of magefiles to determine whether or not the magefile binary
// needs to be rebuilt. This results in faster runtimes, but means that mage
// will fail to rebuild if a dependency has changed. To force a rebuild, run
// mage with the -f flag.
const HashFastEnv = "MAGEFILE_HASHFAST"

// Verbose reports whether a magefile was run with the verbose flag.
func Verbose() bool {
	b, _ := strconv.ParseBool(os.Getenv(VerboseEnv))
	return b
}

// Debug reports whether a magefile was run with the debug flag.
func Debug() bool {
	b, _ := strconv.ParseBool(os.Getenv(DebugEnv))
	return b
}

// GoCmd reports the command that Mage will use to build go code.  By default mage runs
// the "go" binary in the PATH.
func GoCmd() string {
	if cmd := os.Getenv(GoCmdEnv); cmd != "" {
		return cmd
	}
	return "go"
}

// HashFast reports whether the user has requested to use the fast hashing
// mechanism rather than rely on go's rebuilding mechanism.
func HashFast() bool {
	b, _ := strconv.ParseBool(os.Getenv(HashFastEnv))
	return b
}

// IgnoreDefault reports whether the user has requested to ignore the default target
// in the magefile.
func IgnoreDefault() bool {
	b, _ := strconv.ParseBool(os.Getenv(IgnoreDefaultEnv))
	return b
}

// CacheDir returns the directory where mage caches compiled binaries.  It
// defaults to $HOME/.magefile, but may be overridden by the MAGEFILE_CACHE
// environment variable.
func CacheDir() string {
	d := os.Getenv(CacheEnv)
	if d != "" {
		return d
	}
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("HOMEDRIVE"), os.Getenv("HOMEPATH"), "magefile")
	default:
		return filepath.Join(os.Getenv("HOME"), ".magefile")
	}
}

// Namespace allows for the grouping of similar commands
type Namespace struct{}
