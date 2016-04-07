package cfgfile

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/libbeat/logp"
	"gopkg.in/yaml.v2"
)

// Command line flags
var configfile *string
var testConfig *bool

func init() {
	// The default config cannot include the beat name as it is not initialised when this
	// function is called, but see ChangeDefaultCfgfileFlag
	configfile = flag.String("c", "beat.yml", "Configuration file")
	testConfig = flag.Bool("configtest", false, "Test configuration and exit.")
}

// ChangeDefaultCfgfileFlag replaces the value and default value for the `-c` flag so that
// it reflects the beat name.
func ChangeDefaultCfgfileFlag(beatName string) error {
	cliflag := flag.Lookup("c")
	if cliflag == nil {
		return fmt.Errorf("Flag -c not found")
	}

	path, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return fmt.Errorf("Failed to set default config file location because "+
			"the absolute path to %s could not be obtained. %v",
			os.Args[0], err)
	}

	cliflag.DefValue = filepath.Join(path, beatName+".yml")

	return cliflag.Value.Set(cliflag.DefValue)
}

// Read reads the configuration from a yaml file into the given interface structure.
// In case path is not set this method reads from the default configuration file for the beat.
func Read(out interface{}, path string) error {

	if path == "" {
		path = *configfile
	}

	filecontent, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Failed to read %s: %v. Exiting.", path, err)
	}

	filecontent = expandEnv(filecontent)

	if err = yaml.Unmarshal(filecontent, out); err != nil {
		return fmt.Errorf("YAML config parsing failed on %s: %v. Exiting.", path, err)
	}

	return nil
}

func IsTestConfig() bool {
	return *testConfig
}

// expandEnv replaces ${var} in config according to the values of the current
// environment variables. The replacement is case-sensitive. References to
// undefined variables are replaced by the empty string. A default value can be
// given by using the form ${var:default value}.
func expandEnv(config []byte) []byte {
	return []byte(expand(string(config), func(key string) string {
		keyAndDefault := strings.SplitN(key, ":", 2)
		key = keyAndDefault[0]

		v := os.Getenv(key)
		if v == "" && len(keyAndDefault) == 2 {
			// Set value to the default.
			v = keyAndDefault[1]
			logp.Info("Replacing config environment variable '${%s}' with "+
				"default '%s'", key, keyAndDefault[1])
		} else {
			logp.Info("Replacing config environment variable '${%s}' with '%s'",
				key, v)
		}

		return v
	}))
}

// The following methods were copied from the os package of the stdlib. The
// expand method was modified to only expand variables defined with braces and
// ignore $var.

// Expand replaces ${var} in the string based on the mapping function.
func expand(s string, mapping func(string) string) string {
	buf := make([]byte, 0, 2*len(s))
	// ${} is all ASCII, so bytes are fine for this operation.
	i := 0
	for j := 0; j < len(s); j++ {
		if s[j] == '$' && j+2 < len(s) && s[j+1] == '{' {
			buf = append(buf, s[i:j]...)
			name, w := getShellName(s[j+1:])
			buf = append(buf, mapping(name)...)
			j += w
			i = j + 1
		}
	}
	return string(buf) + s[i:]
}

// isShellSpecialVar reports whether the character identifies a special
// shell variable such as $*.
func isShellSpecialVar(c uint8) bool {
	switch c {
	case '*', '#', '$', '@', '!', '?', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return true
	}
	return false
}

// isAlphaNum reports whether the byte is an ASCII letter, number, or underscore
func isAlphaNum(c uint8) bool {
	return c == '_' || '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}

// getShellName returns the name that begins the string and the number of bytes
// consumed to extract it.  If the name is enclosed in {}, it's part of a ${}
// expansion and two more bytes are needed than the length of the name.
func getShellName(s string) (string, int) {
	switch {
	case s[0] == '{':
		if len(s) > 2 && isShellSpecialVar(s[1]) && s[2] == '}' {
			return s[1:2], 3
		}
		// Scan to closing brace
		for i := 1; i < len(s); i++ {
			if s[i] == '}' {
				return s[1:i], i + 1
			}
		}
		return "", 1 // Bad syntax; just eat the brace.
	case isShellSpecialVar(s[0]):
		return s[0:1], 1
	}
	// Scan alphanumerics.
	var i int
	for i = 0; i < len(s) && isAlphaNum(s[i]); i++ {
	}
	return s[:i], i
}
