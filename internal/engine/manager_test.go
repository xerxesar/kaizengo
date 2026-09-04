package engine

import (
	"reflect"
	"testing"
)

func TestSplitInstalledDropsMissingDirs(t *testing.T) {
	present := toSet([]string{"appman", "core", "inventory"})
	keep, drop := splitInstalled([]string{"catalog", "core", "inventory"}, present)
	if !reflect.DeepEqual(keep, []string{"core", "inventory"}) {
		t.Fatalf("keep = %v, want [core inventory]", keep)
	}
	if !reflect.DeepEqual(drop, []string{"catalog"}) {
		t.Fatalf("drop = %v, want [catalog]", drop)
	}
}
