package goproxy

import (
	"net/http"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

var _ Plugin = plugin("")

type plugin string

func (plugin) Module(req *http.Request, prefix string) (Module, error) { panic("implement me") }
func (plugin) Leave(source Module) error                               { panic("implement me") }
func (plugin) Close() error                                            { panic("implement me") }
func (plugin) String() string                                          { return "plugin" }

func nodeLists(n *node) [][]string {
	res := [][]string{}
	if n.f != nil {
		res = append(res, nil)
	}
	for _, f := range n.further {
		for _, ff := range nodeLists(f.node) {
			item := make([]string, len(ff)+1)
			item[0] = f.path
			copy(item[1:], ff)
			res = append(res, item)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		a := res[i]
		b := res[j]
		for k := range a {
			if k >= len(b) {
				return false
			}
			if a[k] != b[k] {
				return a[k] < b[k]
			}
		}
		return false
	})
	return res
}

func TestNode(t *testing.T) {
	n := &node{f: plugin("a")}
	require.Error(t, n.addNode("", plugin("α")))
	if err := n.addNode("gitlab.stageoffice.ru", plugin("b")); err != nil {
		t.Fatal(err)
	}
	if err := n.addNode("gitlab.com", plugin("c")); err != nil {
		t.Fatal(err)
	}
	if err := n.addNode("gitlab.stageoffice.ru/UCS-COMMON/schema", plugin("d")); err != nil {
		t.Fatal(err)
	}
	require.Error(t, n.addNode("gitlab.stageoffice.ru/UCS-COMMON/schema", plugin("δ")))
	if err := n.addNode("gitlab.stageoffice.ru/UCS-CADDY-PLUGINS", plugin("e")); err != nil {
		t.Fatal(err)
	}

	lists := nodeLists(n)
	require.Equal(t, [][]string{
		nil,
		{"gitlab.", "com"},
		{"gitlab.", "stageoffice.ru"},
		{"gitlab.", "stageoffice.ru", "/UCS-C", "ADDY-PLUGINS"},
		{"gitlab.", "stageoffice.ru", "/UCS-C", "OMMON/schema"},
	}, lists)

	// get tests
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "trivial",
			url:      "",
			expected: "a",
		},
		{
			name:     "full-mismatch",
			url:      "github.com/sirkon/goproxy",
			expected: "a",
		},
		{
			name:     "to-the-gitlab-com",
			url:      "gitlab.com/repo/project",
			expected: "c",
		},
		{
			name:     "stageoffice-generic",
			url:      "gitlab.stageoffice.ru/UCS-PLATFORM/marker",
			expected: "b",
		},
		{
			name:     "stageoffice-schema",
			url:      "gitlab.stageoffice.ru/UCS-COMMON/schema/marker",
			expected: "d",
		},
		{
			name:     "stageoffice-caddy-plugins",
			url:      "gitlab.stageoffice.ru/UCS-CADDY-PLUGINS/algol",
			expected: "e",
		},
		{
			name:     "match-rollback",
			url:      "gitlab.org",
			expected: "a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := n.getNode(tt.url)
			if string(res.(plugin)) != tt.expected {
				t.Errorf("plugin %v expected for url %s, got %v", plugin(tt.expected), tt.url, res)
			}
		})
	}

	if err := n.addNode("gitlab.stageoffice.ru/UCS-PLATFORM/marker", plugin("f")); err != nil {
		t.Fatal(err)
	}
	if err := n.addNode("gitlab.stageoffice.ru/UCS-PLATFORM", plugin("g")); err != nil {
		t.Fatal(err)
	}
	lists = nodeLists(n)
	require.Equal(t, [][]string{
		nil,
		{"gitlab.", "com"},
		{"gitlab.", "stageoffice.ru"},
		{"gitlab.", "stageoffice.ru", "/UCS-", "C", "ADDY-PLUGINS"},
		{"gitlab.", "stageoffice.ru", "/UCS-", "C", "OMMON/schema"},
		{"gitlab.", "stageoffice.ru", "/UCS-", "PLATFORM"},
		{"gitlab.", "stageoffice.ru", "/UCS-", "PLATFORM", "/marker"},
	}, lists)

	if err := n.addNode("somehost.com", plugin("y")); err != nil {
		t.Fatal(err)
	}
	lists = nodeLists(n)
	require.Equal(t, [][]string{
		nil,
		{"gitlab.", "com"},
		{"gitlab.", "stageoffice.ru"},
		{"gitlab.", "stageoffice.ru", "/UCS-", "C", "ADDY-PLUGINS"},
		{"gitlab.", "stageoffice.ru", "/UCS-", "C", "OMMON/schema"},
		{"gitlab.", "stageoffice.ru", "/UCS-", "PLATFORM"},
		{"gitlab.", "stageoffice.ru", "/UCS-", "PLATFORM", "/marker"},
		{"somehost.com"},
	}, lists)

	if err := n.addNode("gitlab.stageoffice.ru/UCS-C", plugin("z")); err != nil {
		t.Fatal(err)
	}
	lists = nodeLists(n)
	require.Equal(t, [][]string{
		nil,
		{"gitlab.", "com"},
		{"gitlab.", "stageoffice.ru"},
		{"gitlab.", "stageoffice.ru", "/UCS-", "C"},
		{"gitlab.", "stageoffice.ru", "/UCS-", "C", "ADDY-PLUGINS"},
		{"gitlab.", "stageoffice.ru", "/UCS-", "C", "OMMON/schema"},
		{"gitlab.", "stageoffice.ru", "/UCS-", "PLATFORM"},
		{"gitlab.", "stageoffice.ru", "/UCS-", "PLATFORM", "/marker"},
		{"somehost.com"},
	}, lists)
}
