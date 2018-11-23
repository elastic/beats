package ilm

//func TestNewIlmPolicyCfg(t *testing.T) {
//	beatInfo := beat.Info{
//		Beat:        "testbeatilm",
//		IndexPrefix: "testbeat",
//		Version:     "1.2.3",
//	}
//	pName := "deleteAfter30days"
//
//	for _, data := range []struct{
//		idx string
//		cfg *ilmPolicyCfg
//	}{
//		{"", nil},
//		{"testbeat", &ilmPolicyCfg{idxName: "testbeat", policyName: pName}},
//		{"testbeat-%{[beat.version]}", &ilmPolicyCfg{idxName: "testbeat-1.2.3", policyName: pName}},
//		{"testbeat-SNAPSHOT-%{[beat.version]}", &ilmPolicyCfg{idxName: "testbeat-snapshot-1.2.3", policyName: pName}},
//		{"testbeat-%{[beat.version]}-%{+yyyy.MM.dd}", nil},
//	}{
//		cfg := newIlmPolicyCfg(data.idx, pName, beatInfo)
//		assert.Equal(t, data.cfg, cfg)
//	}
//}
