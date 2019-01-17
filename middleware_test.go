package goproxy

import "testing"

func Test_getVersion(t *testing.T) {
	type args struct {
		suffix string
	}
	tests := []struct {
		name   string
		suffix string
		want   string
	}{
		{
			name:   "zip",
			suffix: "v0.1.2.zip",
			want:   "v0.1.2",
		},
		{
			name:   "info",
			suffix: "v0.1.2.info",
			want:   "v0.1.2",
		},
		{
			name:   "mod",
			suffix: "v0.1.2.mod",
			want:   "v0.1.2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getVersion(tt.suffix); got != tt.want {
				t.Errorf("getVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
