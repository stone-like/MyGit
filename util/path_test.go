package util

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_Descend(t *testing.T) {

	for _, d := range []struct {
		title      string
		expected   []string
		targetPath string
	}{
		{"absoluteParh",
			[]string{"/", "/some", "/some/dummy", "/some/dummy/path"},
			"/some/dummy/path"},
		{
			"relativePath",
			[]string{"some", "some/dummy", "some/dummy/path"},
			"some/dummy/path",
		},
	} {

		t.Run(d.title, func(t *testing.T) {
			s := Descend(d.targetPath)

			if diff := cmp.Diff(d.expected, s); diff != "" {
				t.Errorf("diff is: %s\n", diff)
			}
		})
	}
}
