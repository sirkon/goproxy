package router

import (
	"github.com/sirkon/goproxy/source"
)

// Router routes to some factory
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

// AddRoute add factory for a given path mask
func (r *Router) AddRoute(mask string, f source.Factory) error {
	return r.tree.addNode(mask, f)
}

// Factory returns factory for given route
func (r *Router) Factory(path string) source.Factory {
	return r.tree.getNode(path)
}
