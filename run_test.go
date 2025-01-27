package gostata

import (
	"path"
	"testing"
)

const (
	testDir = "/Users/drgo/local/code/gostata/testing"
)

func getTestingPath(filename string)string {
	return path.Join(testDir, filename)
}

func TestRunStata(t *testing.T) {
	output, err := RunStataDo(testDir, "do.do")
	if err != nil {
		t.Fatalf("%s", err)
	}
	t.Logf("out: %s", output)
	dict := GetKeyValuePairs(output)
	t.Logf("dict: %v", dict)
}
