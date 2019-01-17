package router

import (
	"github.com/sirkon/goproxy/source"
)

// Router routes to some plugin
// TODO: implement prefix tree based router
type Router struct {
	tree *node
}

// NewRouter ...
func NewRouter() (*Router, error) {
	return &Router{
		tree: &node{},
	}, nil
}

// AddRoute add plugin for a given path mask
func (r *Router) AddRoute(mask string, f source.Plugin) error {
	return r.tree.addNode(mask, f)
}

// Plugin returns plugin for given route
func (r *Router) Factory(path string) source.Plugin {
	return r.tree.getNode(path)
}
