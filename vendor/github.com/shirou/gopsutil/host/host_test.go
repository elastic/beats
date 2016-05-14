package host

import (
	"fmt"
	"testing"
)

func TestHostInfo(t *testing.T) {
	v, err := Info()
	if err != nil {
		t.Errorf("error %v", err)
	}
	empty := &InfoStat{}
	if v == empty {
		t.Errorf("Could not get hostinfo %v", v)
	}
}

func TestBoot_time(t *testing.T) {
	v, err := BootTime()
	if err != nil {
		t.Errorf("error %v", err)
	}
	if v == 0 {
		t.Errorf("Could not get boot time %v", v)
	}
}

func TestUsers(t *testing.T) {
	v, err := Users()
	if err != nil {
		t.Errorf("error %v", err)
	}
	empty := UserStat{}
	for _, u := range v {
		if u == empty {
			t.Errorf("Could not Users %v", v)
		}
	}
}

func TestHostInfoStat_String(t *testing.T) {
	v := InfoStat{
		Hostname: "test",
		Uptime:   3000,
		Procs:    100,
		OS:       "linux",
		Platform: "ubuntu",
		BootTime: 1447040000,
	}
	e := `{"hostname":"test","uptime":3000,"bootTime":1447040000,"procs":100,"os":"linux","platform":"ubuntu","platformFamily":"","platformVersion":"","virtualizationSystem":"","virtualizationRole":""}`
	if e != fmt.Sprintf("%v", v) {
		t.Errorf("HostInfoStat string is invalid: %v", v)
	}
}

func TestUserStat_String(t *testing.T) {
	v := UserStat{
		User:     "user",
		Terminal: "term",
		Host:     "host",
		Started:  100,
	}
	e := `{"user":"user","terminal":"term","host":"host","started":100}`
	if e != fmt.Sprintf("%v", v) {
		t.Errorf("UserStat string is invalid: %v", v)
	}
}
