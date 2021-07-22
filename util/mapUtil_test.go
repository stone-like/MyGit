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
