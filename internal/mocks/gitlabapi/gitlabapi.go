package gitlabapi

import (
	"context"
	"io"

	"github.com/sirkon/gitlab"
	"github.com/sirkon/gitlab/gitlabdata"
	"github.com/stretchr/testify/mock"
)

var _ gitlab.Client = &GitlabAPICLient{}

type GitlabAPICLient struct {
	mock.Mock
}

func (g *GitlabAPICLient) Tags(ctx context.Context, project, tagPrefix string) ([]*gitlabdata.Tag, error) {
	res := g.Called(project, tagPrefix)
	return res.Get(0).([]*gitlabdata.Tag), res.Error(1)
}

func (g *GitlabAPICLient) File(ctx context.Context, project, path, ref string) ([]byte, error) {
	res := g.Called(project, ref)
	return res.Get(0).([]byte), res.Error(1)
}

func (g *GitlabAPICLient) ProjectInfo(ctx context.Context, project string) (*gitlabdata.Project, error) {
	res := g.Called(project)
	return res.Get(0).(*gitlabdata.Project), res.Error(1)
}

func (g *GitlabAPICLient) Archive(ctx context.Context, projectID int, ref string) (io.ReadCloser, error) {
	res := g.Called(projectID, ref)
	return res.Get(0).(io.ReadCloser), res.Error(1)
}

func (g *GitlabAPICLient) Commits(ctx context.Context, project string, ref string) ([]*gitlabdata.Commit, error) {
	res := g.Called(project, ref)
	return res.Get(0).([]*gitlabdata.Commit), res.Error(1)
}
