package util

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_SortKeyAndRerverse(t *testing.T) {

	for _, d := range []struct {
		title        string
		expectedKeys []string
		targetMap    map[string]struct{}
		targetFunc   func(m map[string]struct{}) []string
	}{
		{"sortedKeys", []string{"aaa", "aaa/bbb", "aaa/bbb/ccc", "aaa/xxx", "aaa/xxx/yyy"}, map[string]struct{}{"aaa": {}, "aaa/xxx": {}, "aaa/xxx/yyy": {}, "aaa/bbb": {}, "aaa/bbb/ccc": {}},
			func(m map[string]struct{}) []string {
				return SortedKeys(m)
			}},
		{"sortedKeysReverse", []string{"aaa/xxx/yyy", "aaa/xxx", "aaa/bbb/ccc", "aaa/bbb", "aaa"}, map[string]struct{}{"aaa": {}, "aaa/xxx": {}, "aaa/xxx/yyy": {}, "aaa/bbb": {}, "aaa/bbb/ccc": {}},
			func(m map[string]struct{}) []string {
				return SortKeysReverse(SortedKeys(m))
			}},
	} {
		t.Run(d.title, func(t *testing.T) {
			ret := d.targetFunc(d.targetMap)
			if diff := cmp.Diff(d.expectedKeys, ret); diff != "" {
				t.Errorf("diff is: %s\n", diff)
			}
		})
	}
}

func TestMapCopy(t *testing.T) {

	m := make(map[string]int)

	test := map[string]int{
		"test1": 1,
		"test2": 2,
	}

	Copy(m, test)

	if diff := cmp.Diff(m, test); diff != "" {
		t.Errorf("diff is: %s\n", diff)
	}

	m2 := make(map[string]string)

	test2 := map[string]string{
		"test1": "11",
		"test2": "22",
	}

	Copy(m2, test2)

	if diff := cmp.Diff(m2, test2); diff != "" {
		t.Errorf("diff is: %s\n", diff)
	}

	m3 := make(map[int]string)

	test3 := map[int]string{
		1: "11",
		2: "22",
	}

	Copy(m3, test3)

	if diff := cmp.Diff(m3, test3); diff != "" {
		t.Errorf("diff is: %s\n", diff)
	}
}
