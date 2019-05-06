package semver

import "testing"

func TestMajor(t *testing.T) {
	tests := []struct {
		name   string
		sample string
		want   int
	}{
		{
			name:   "sample-1",
			sample: "v12.3.1",
			want:   12,
		},
		{
			name:   "sample-2",
			sample: "v0.0.1",
			want:   0,
		},
		{
			name:   "sample-3",
			sample: "v.1.2",
			want:   -1,
		},
		{
			name:   "sample-4",
			sample: "",
			want:   -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Major(tt.sample); got != tt.want {
				t.Errorf("Major() = %v, want %v", got, tt.want)
			}
		})
	}
}
