package goproxy

import (
	"strings"

	"github.com/sirkon/goproxy/internal/errors"
)

type nodeExtension struct {
	path string
	node *node
}

type node struct {
	f       Plugin
	further []*nodeExtension
}

func (n *node) addNode(path string, f Plugin) error {
	return n.realAdd(path, path, f)
}

func (n *node) getNode(path string) Plugin {
	return n.realGet(path, path)
}

func (n *node) realAdd(path string, origPath string, f Plugin) error {
	if len(path) == 0 {
		if n.f == nil {
			n.f = f
		} else {
			return errors.Newf("cannot prolong a node with given path %s as it was taken before", origPath)
		}
		return nil
	}

	for _, ne := range n.further {
		switch {
		case strings.HasPrefix(path, ne.path):
			return ne.node.realAdd(path[len(ne.path):], origPath, f)
		case strings.HasPrefix(ne.path, path):
			// decompose a path
			tail := ne.path[len(path):]
			newNode := &node{
				f: f,
				further: []*nodeExtension{
					{
						path: tail,
						node: ne.node,
					},
				},
			}
			ne.path = path
			ne.node = newNode
			return nil
		default:
			cp := commonPrefix(path, ne.path)
			if len(cp) > 0 {
				tail1 := path[len(cp):]
				tail2 := ne.path[len(cp):]
				newNode := &node{
					further: []*nodeExtension{
						{
							path: tail1,
							node: &node{
								f: f,
							},
						},
						{
							path: tail2,
							node: ne.node,
						},
					},
				}
				ne.path = cp
				ne.node = newNode
				return nil
			}
		}
	}

	n.further = append(n.further, &nodeExtension{
		path: path,
		node: &node{
			f: f,
		},
	})

	return nil
}

func commonPrefix(p1 string, p2 string) string {
	for i := range p1 {
		// it is clear we will stop before the end of either strings
		if p1[i] != p2[i] {
			return p1[:i]
		}
	}
	return ""
}

func (n *node) realGet(path string, origPath string) Plugin {
	for _, ne := range n.further {
		if strings.HasPrefix(path, ne.path) {
			res := ne.node.realGet(path[len(ne.path):], origPath)
			if res == nil {
				return n.f
			}
			return res
		}
	}
	return n.f
}
