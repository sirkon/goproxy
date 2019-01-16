package gitlab

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

// Client implements access for gitlab functions needed
type Client interface {
	Tags(ctx context.Context, project, tag, token string) ([]byte, error)
	GoMod(ctx context.Context, project, tag, token string) ([]byte, error)
	ModuleInfo(ctx context.Context, project, token string) ([]byte, error)
	Archive(ctx context.Context, project, tag, token string) (io.ReadCloser, error)
}

// NewClient returns direct gitlab client
func NewClient(url string, client *http.Client) Client {
	return &gitlabClient{
		url:    url,
		client: client,
	}
}

type gitlabClient struct {
	url    string
	client *http.Client
}

func (c *gitlabClient) Tags(ctx context.Context, project, tag, token string) ([]byte, error) {
	urlPath := fmt.Sprintf("/projects/%s/repository/tags", url.PathEscape(project))
	if len(tag) > 0 {
		urlPath += "/" + tag
	}
	resp, err := c.makeRequest(ctx, urlPath, token, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func (c *gitlabClient) GoMod(ctx context.Context, project, tag, token string) ([]byte, error) {
	resp, err := c.makeRequest(ctx, fmt.Sprintf("/projects/%s/repository/files/go.mod", url.PathEscape(project)), token, map[string]string{
		"ref": tag,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	type responseType struct {
		Encoding      string `json:"encoding"`
		Content       string `json:"content"`
		ContentSHA256 string `json:"content_sha256"`
	}
	dec := json.NewDecoder(resp.Body)
	var response responseType

	if err := dec.Decode(&response); err != nil {
		return nil, err
	}

	var content []byte
	switch response.Encoding {
	case "base64":
		content, err = base64.StdEncoding.DecodeString(response.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to decode go.mod content %s: %s", response.Content, err)
		}
	default:
		return nil, fmt.Errorf("encoding %s is not supported: %s", response.Encoding)
	}

	return content, nil
}

func (c *gitlabClient) ModuleInfo(ctx context.Context, project, token string) ([]byte, error) {
	resp, err := c.makeRequest(ctx, fmt.Sprintf("/projects/%s", url.PathEscape(project)), token, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func (c *gitlabClient) Archive(ctx context.Context, project, tag, token string) (io.ReadCloser, error) {
	resp, err := c.makeRequest(ctx, fmt.Sprintf("/projects/%s/repository/archive.zip", url.PathEscape(project)), token, map[string]string{
		"ref": tag,
	})
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (c *gitlabClient) makeRequest(ctx context.Context, urlValue string, token string, keys map[string]string) (resp *http.Response, err error) {
	v := url.Values{}
	v.Set("private_token", token)
	for key, value := range keys {
		v.Set(key, value)
	}

	requestURL := c.url + urlValue + "?" + v.Encode()
	log.Println("requesting", requestURL)
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate request: %s", err)
	}
	req = req.WithContext(ctx)
	resp, err = c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get response: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		res, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("error response (Status Code %d, body %s)", resp.StatusCode, string(res))
	}
	return resp, nil
}
