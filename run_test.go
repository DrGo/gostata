package gostata

import (
	"testing"
)

const (
	testDir = "/Users/drgo/local/code/gostata/testing"
)

func TestRunStata(t *testing.T) {
	output, err := RunStataDo(testDir, "do.do")
	if err != nil {
		t.Fatalf("%s", err)
	}
	t.Logf("out: %s", output)
	dict := GetKeyValuePairs(output)
	t.Logf("dict: %v", dict)
}
