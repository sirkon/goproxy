package gitlab

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/sirkon/gitlab/gitlabdata"
	"github.com/stretchr/testify/mock"

	"github.com/sirkon/goproxy/internal/mocks/gitlabapi"
)

func Test_gitlabModule_Versions(t *testing.T) {
	client1 := &gitlabapi.GitlabAPICLient{}
	client1.On("Tags", "github.com/user/project", "").Return(
		[]*gitlabdata.Tag{
			{
				Commit: &gitlabdata.Commit{
					ID:        "1",
					ShortID:   "1",
					CreatedAt: time.RFC3339,
				},
				Release: nil,
				Name:    "v0.0.1",
			},
			{
				Commit: &gitlabdata.Commit{
					ID:        "1",
					ShortID:   "1",
					CreatedAt: time.RFC3339,
				},
				Release: nil,
				Name:    "v0.0.2",
			},
			{
				Commit: &gitlabdata.Commit{
					ID:        "1",
					ShortID:   "1",
					CreatedAt: time.RFC3339,
				},
				Release: nil,
				Name:    "v0.0.3",
			},
		},
		nil,
	)

	tests := []struct {
		name    string
		gitlab  *gitlabModule
		prefix  string
		want    []string
		wantErr bool
	}{
		{
			name: "test-1",
			gitlab: &gitlabModule{
				client:          client1,
				fullPath:        "github.com/user/project",
				path:            "github.com/user/project",
				pathUnversioned: "github.com/user/project",
			},
			prefix:  "",
			want:    []string{"v0.0.1", "v0.0.2", "v0.0.3"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.gitlab.Versions(context.Background(), tt.prefix)
			if (err != nil) != tt.wantErr {
				t.Errorf("Versions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Versions() got = %v, want %v", got, tt.want)
			}
		})
	}

	mock.AssertExpectationsForObjects(t, client1)
}
