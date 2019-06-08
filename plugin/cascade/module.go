package cascade

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/rs/zerolog"

	"github.com/sirkon/goproxy"
)

var _ goproxy.Module = &cascadeModule{}

type cascadeModule struct {
	mod       string
	reqMod    string
	url       string
	client    *http.Client
	basicAuth struct {
		ok       bool
		user     string
		password string
	}
}

func (s *cascadeModule) ModulePath() string {
	return s.mod
}

func (s *cascadeModule) Versions(ctx context.Context, prefix string) (tags []string, err error) {
	resp, err := s.makeRequest(ctx, fmt.Sprintf("%s/%s/@v/list", s.url, s.reqMod))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read out list request: %s", err)
	}

	var res []string
	for _, version := range strings.Split(string(data), "\n") {
		version = strings.TrimSpace(version)
		if len(version) > 0 {
			res = append(res, version)
		}
	}
	return res, nil
}

func (s *cascadeModule) Stat(ctx context.Context, rev string) (*goproxy.RevInfo, error) {
	resp, err := s.makeRequest(ctx, fmt.Sprintf("%s/%s/@v/%s.info", s.url, s.reqMod, rev))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var dest goproxy.RevInfo
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&dest); err != nil {
		return nil, fmt.Errorf("failed to decode stat data for %s: %s", s.reqMod, err)
	}

	return &dest, nil
}

func (s *cascadeModule) GoMod(ctx context.Context, version string) (data []byte, err error) {
	resp, err := s.makeRequest(ctx, fmt.Sprintf("%s/%s/@v/%s.mod", s.url, s.reqMod, version))
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

func (s *cascadeModule) Zip(ctx context.Context, version string) (file io.ReadCloser, err error) {
	resp, err := s.makeRequest(ctx, fmt.Sprintf("%s/%s/@v/%s.zip", s.url, s.reqMod, version))
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (s *cascadeModule) makeRequest(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate request to %s: %s", url, err)
	}
	if s.basicAuth.ok {
		req.SetBasicAuth(s.basicAuth.user, s.basicAuth.password)
	}
	req = req.WithContext(ctx)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get response from %s: %s", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		defer func() {
			if err := resp.Body.Close(); err != nil {
				zerolog.Ctx(ctx).Error().Err(err).Msgf("failed to close response body from %s", url)
			}
		}()
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msgf("failed to get response data from %s", url)
			return nil, err
		}
		zerolog.Ctx(ctx).Error().Msgf("unexpected status code %d from %s: \n%s", resp.StatusCode, url, string(data))
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	return resp, nil
}
