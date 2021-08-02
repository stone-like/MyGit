package src

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func TestPrintConflictStatusLong(t *testing.T) {
	s := GenerateStatus()

	for _, d := range []struct {
		title          string
		expectedString string
		targetPath     string
		targetSlice    []int
	}{
		{
			"both modified",
			"Unmerged paths:\n\t   both modified:hello.txt\n",
			"hello.txt",
			[]int{1, 2, 3},
		},
		{
			"deleted by them:",
			"Unmerged paths:\n\t deleted by them:hello.txt\n",
			"hello.txt",
			[]int{1, 2},
		},
		{
			"deleted by us:",
			"Unmerged paths:\n\t   deleted by us:hello.txt\n",
			"hello.txt",
			[]int{1, 3},
		},
		{
			"both added:",
			"Unmerged paths:\n\t      both added:hello.txt\n",
			"hello.txt",
			[]int{2, 3},
		},
		{
			"added by us:",
			"Unmerged paths:\n\t     added by us:hello.txt\n",
			"hello.txt",
			[]int{2},
		},
		{
			"added by them:",
			"Unmerged paths:\n\t   added by them:hello.txt\n",
			"hello.txt",
			[]int{3},
		},
	} {
		t.Run(d.title, func(t *testing.T) {
			s.Conflicts[d.targetPath] = d.targetSlice

			var buf bytes.Buffer

			err := s.GenerateChangesMessage(ConflictedMessage, s.Conflicts, &buf)
			assert.NoError(t, err)

			str := buf.String()

			if diff := cmp.Diff(d.expectedString, str); diff != "" {
				t.Errorf("diff is %s\n", diff)
			}
		})
	}

}

func TestPrintConflictStatusShort(t *testing.T) {

	for _, d := range []struct {
		title          string
		expectedString string
		targetPath     string
		targetSlice    []int
	}{
		{
			"both modified",
			"UU hello.txt\n",
			"hello.txt",
			[]int{1, 2, 3},
		},
		{
			"deleted by them:",
			"UD hello.txt\n",
			"hello.txt",
			[]int{1, 2},
		},
		{
			"deleted by us:",
			"DU hello.txt\n",
			"hello.txt",
			[]int{1, 3},
		},
		{
			"both added:",
			"AA hello.txt\n",
			"hello.txt",
			[]int{2, 3},
		},
		{
			"added by us:",
			"AU hello.txt\n",
			"hello.txt",
			[]int{2},
		},
		{
			"added by them:",
			"UA hello.txt\n",
			"hello.txt",
			[]int{3},
		},
	} {
		t.Run(d.title, func(t *testing.T) {
			s := GenerateStatus()

			s.Changed = append(s.Changed, d.targetPath)
			s.Conflicts[d.targetPath] = d.targetSlice

			var buf bytes.Buffer

			err := s.WritePorcelainStatus(&buf)
			assert.NoError(t, err)

			str := buf.String()

			if diff := cmp.Diff(d.expectedString, str); diff != "" {
				t.Errorf("diff is %s\n", diff)
			}
		})
	}

}
