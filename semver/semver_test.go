package semver

import (
	"testing"
)

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

func TestMajorMinorPatch(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    [3]int
	}{
		{
			name:    "trivial",
			version: "v1.2.3",
			want:    [3]int{1, 2, 3},
		},
		{
			name:    "valid",
			version: "v1.2.3-alpha",
			want:    [3]int{1, 2, 3},
		},
		{
			name:    "another-valid",
			version: "v1.2.3+alpha",
			want:    [3]int{1, 2, 3},
		},
		{
			name:    "invalid",
			version: "v1.2-alpha",
			want:    [3]int{0, 0, 0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2 := MajorMinorPatch(tt.version)
			if got != tt.want[0] {
				t.Errorf("MajorMinorPatch() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want[1] {
				t.Errorf("MajorMinorPatch() got1 = %v, want %v", got1, tt.want[1])
			}
			if got2 != tt.want[2] {
				t.Errorf("MajorMinorPatch() got2 = %v, want %v", got2, tt.want[2])
			}
		})
	}
}

func TestPseudo(t *testing.T) {
	tests := []struct {
		name string
		v    string
		want string
	}{
		{
			name: "simple-semver",
			v:    "v0.1.2",
			want: "",
		},
		{
			name: "semver-with-suffix",
			v:    "v0.1.2-alpha",
			want: "",
		},
		{
			name: "pseudo-semver",
			v:    "v0.0.0-20190313170020-28fc84874d7f",
			want: "28fc84874d7f",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Pseudo(tt.v); got != tt.want {
				t.Errorf("Pseudo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsPrerelease(t *testing.T) {
	tests := []struct {
		name string
		v    string
		want bool
	}{
		{
			name: "not-1",
			v:    "v0.1.2",
			want: false,
		},
		{
			name: "not-2",
			v:    "v0.0.0-20190313170020-28fc84874d7f",
			want: false,
		},
		{
			name: "alpha",
			v:    "v2.1.2-pre-meta",
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPrerelease(tt.v); got != tt.want {
				t.Errorf("IsPrerelease() = %v, want %v", got, tt.want)
			}
		})
	}
}
