package src

import (
	"bytes"
	"mygit/src/database/content"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLogConflict(t *testing.T) {
	m := &Merge{
		leftName:  "leftBranch",
		rightName: "rightBranch",
	}

	for _, d := range []struct {
		title     string
		conflicts map[string][]*content.Entry
		rename    string
		expected  string
	}{
		{
			"LogLeftRightConflict content",
			map[string][]*content.Entry{
				"a.txt": {&content.Entry{}, &content.Entry{}, &content.Entry{}},
			},
			"",
			"CONFLICT (content): Merge conflict in a.txt\n",
		},
		{
			"LogLeftRightConflict add/add",
			map[string][]*content.Entry{
				"a.txt": {nil, &content.Entry{}, &content.Entry{}},
			},
			"",
			"CONFLICT (add/add): Merge conflict in a.txt\n",
		},
		{
			"LogModifyDeleteConflict modified on Right And Deleted on Left",
			map[string][]*content.Entry{
				"a.txt": {&content.Entry{}, nil, &content.Entry{}},
			},
			"",
			"CONFLICT (modify/delete): a.txt deleted in leftBranch and modified in rightBranch.\nVersion rightBranch of a.txt left in tree\n",
		},
		{
			"LogModifyDeleteConflict modified on Left And Deleted on Right",
			map[string][]*content.Entry{
				"a.txt": {&content.Entry{}, &content.Entry{}, nil},
			},
			"",
			"CONFLICT (modify/delete): a.txt deleted in rightBranch and modified in leftBranch.\nVersion leftBranch of a.txt left in tree\n",
		},
		{
			"LogModifyDeleteConflict with Rename",
			map[string][]*content.Entry{
				"a.txt": {&content.Entry{}, &content.Entry{}, nil},
			},
			"RenameChanged",
			"CONFLICT (modify/delete): a.txt deleted in rightBranch and modified in leftBranch.\nVersion leftBranch of a.txt left in tree at RenameChanged\n",
		},
		{
			"LogFileDirConflict directory/file",
			map[string][]*content.Entry{
				"a.txt": {nil, &content.Entry{}, nil},
			},
			"Changed",
			"Conflict (directory/file): There is a directory with name a.txt in leftBranch.\nAdding a.txt as Changed\n",
		},
		{
			"LogFileDirConflict file/directory",
			map[string][]*content.Entry{
				"a.txt": {nil, nil, &content.Entry{}},
			},
			"Changed",
			"Conflict (file/directory): There is a directory with name a.txt in rightBranch.\nAdding a.txt as Changed\n",
		},
	} {
		t.Run(d.title, func(t *testing.T) {
			var buf bytes.Buffer
			rm := GenerateResolveMerge(m, &buf)
			rm.conflicts = d.conflicts

			rm.RunLogConflict("a.txt", d.rename)

			str := buf.String()

			if diff := cmp.Diff(d.expected, str); diff != "" {
				t.Errorf("diff is %s\n", diff)
			}
		})
	}

}
