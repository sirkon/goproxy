package gomod_test

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sirkon/goproxy/gomod"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		want     *gomod.Module
		wantErr  bool
	}{
		{
			name:     "test",
			fileName: "testdata/go.mod.golden",
			want: &gomod.Module{
				Name:      "github.com/sirkon/goproxy",
				GoVersion: "1.13",
				Require: map[string]string{
					"github.com/davecgh/go-spew": "v1.1.1",
					"github.com/rs/zerolog":      "v1.14.3",
					"github.com/sirkon/message":  "v1.5.1",
				},
				Exclude: map[string]string{
					"github.com/davecgh/go-spew": "v1.1.3",
				},
				Replace: map[string]gomod.Replacement{
					"github.com/rs/zerolog": gomod.RelativePath("../../../github.com/rs/zerolog"),
					"github.com/sirkon/message": gomod.Dependency{
						Path:    "github.com/sirkon/message",
						Version: "v1.5.2",
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := ioutil.ReadFile(tt.fileName)
			if err != nil {
				t.Error(err)
			}
			got, err := gomod.Parse(tt.fileName, input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, got, tt.want)
		})
	}
}
