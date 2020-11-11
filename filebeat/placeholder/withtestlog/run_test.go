package withtestlog

import "testing"

func TestLogIsPrintedOnError(t *testing.T) {
	t.Log("Log message should be printed")
	t.Logf("printf style log message: %v", 42)
	t.Error("Log should fail")
	t.Errorf("Log should fail with printf style log: %v", 23)
}

func TestLogIsPrintedOnFatal(t *testing.T) {
	t.Log("Log message should be printed")
	t.Logf("printf style log message: %v", 42)
	t.Fatal("Log should fail")
}

func TestLogIsPrintedOnFatalf(t *testing.T) {
	t.Log("Log message should be printed")
	t.Logf("printf style log message: %v", 42)
	t.Fatalf("Log should fail with printf style log: %v", 42)
}

func TestLogsWithNewlines(t *testing.T) {
	t.Log("Log\nmessage\nshould\nbe\nprinted")
	t.Logf("printf\nstyle\nlog\nmessage:\n%v", 42)
	t.Fatalf("Log\nshould\nfail\nwith\nprintf\nstyle\nlog:\n%v", 42)
}
