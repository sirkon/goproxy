package cascade

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/sirkon/goproxy/source"
)

var _ source.Source = &cascadeSource{}

type cascadeSource struct {
	mod    string
	url    string
	client *http.Client
}

func (s *cascadeSource) ModulePath() string {
	return s.mod
}

func (s *cascadeSource) Versions(ctx context.Context, prefix string) (tags []string, err error) {
	resp, err := s.makeRequest(ctx, fmt.Sprintf("%s/%s/@v/list", s.url, s.mod))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read out list request: %s", err)
	}

	return strings.Split(string(data), "\n"), nil
}

func (s *cascadeSource) Stat(ctx context.Context, rev string) (*source.RevInfo, error) {
	resp, err := s.makeRequest(ctx, fmt.Sprintf("%s/%s/@v/%s.info", s.url, s.mod, rev))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var dest source.RevInfo
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&dest); err != nil {
		return nil, fmt.Errorf("failed to decode stat data for %s: %s", s.mod, err)
	}

	return &dest, nil
}

func (s *cascadeSource) GoMod(ctx context.Context, version string) (data []byte, err error) {
	resp, err := s.makeRequest(ctx, fmt.Sprintf("%s/%s/@v/%s.mod", s.url, s.mod, version))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read out mod request for %s: %s", s.mod, err)
	}

	return
}

func (s *cascadeSource) Zip(ctx context.Context, version string) (file io.ReadCloser, err error) {
	resp, err := s.makeRequest(ctx, fmt.Sprintf("%s/%s/@v/%s.zip", s.url, s.mod, version))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return resp.Body, nil
}

func (s *cascadeSource) makeRequest(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate request on %s: %s", url, err)
	}

	req = req.WithContext(ctx)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get response from %s: %s", url, err)
	}

	return resp, nil
}
