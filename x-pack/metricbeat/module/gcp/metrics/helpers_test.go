package metrics

import "testing"

func Test_withSuffix(t *testing.T) {
	type args struct {
		s      string
		suffix string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "when suffix already present",
			args: args{"foo/", "/"},
			want: "foo/",
		},
		{
			name: "when suffix missing",
			args: args{"foo", "/"},
			want: "foo/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := withSuffix(tt.args.s, tt.args.suffix); got != tt.want {
				t.Errorf("withSuffix() = %v, want %v", got, tt.want)
			}
		})
	}
}
