package thelpers

import (
	"strings"
)

// TestingLikeOutput emulates
type TestingLikeOutput interface {
	Log(args ...interface{})
}

type OutputWriter struct {
	T TestingLikeOutput
}

func (w OutputWriter) Write(p []byte) (n int, err error) {
	w.T.Log("\r" + strings.TrimRight(string(p), "\n"))
	return len(p), nil
}
