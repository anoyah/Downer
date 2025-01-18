package tools

import "testing"

func TestGenID(t *testing.T) {
	id := GenLayerID("parent id", "current id")
	if id == "" {
		t.Fail()
	}
}
