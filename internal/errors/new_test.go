package errors

import (
	"testing"
)

func TestNew(t *testing.T) {
	err := New("error")
	if err.Error() != "error" {
		t.Errorf("error expected, got %s", err.Error())
	}
}

func TestNewf(t *testing.T) {
	err := Newf("error: %d", 1)
	if err.Error() != "error: 1" {
		t.Errorf("`error: 1` expected, got %s", err.Error())
	}
}
