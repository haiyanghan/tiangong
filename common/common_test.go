package common_test

import (
	"github.com/haiyanghan/tiangong/common"
	"testing"
)

func TestEmpty(t *testing.T) {
	var a interface{} = struct {
	}{}
	if common.IsEmpty(a) {
		t.Error()
		return
	}

	table := make(map[string]string)
	if common.IsNotEmpty(table) {
		t.Error()
		return
	}
	table["tiangong"] = "1"
	if common.IsEmpty(table) {
		t.Error()
		return
	}
}
