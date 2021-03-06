package src

import (
	"fmt"
	"io"
	"mygit/src/crypt"
	data "mygit/src/database"
	con "mygit/src/database/content"
	"mygit/util"
	"path/filepath"
	"strings"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
)

type DiffOption struct {
	Cached bool
	Stage  int
}

func StartDiff(w io.Writer, rootPath string, option *DiffOption) error {
	gitPath := filepath.Join(rootPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(rootPath, gitPath, dbPath)

	err := repo.i.Load()

	if err != nil {
		return err
	}

	s := GenerateStatus()
	err = s.IntitializeStatus(repo)
	if err != nil {
		return err
	}

	if option.Cached {
		//Index<->CommitHead
		err = DiffHeadIndex(s, repo, w)
		if err != nil {
			return err
		}
	} else {
		//Index<->Workspace
		err = DiffIndexWorkSpace(option.Stage, s, repo, w)
		if err != nil {
			return err
		}
	}

	return nil
}

var (
	NULLObjId = strings.Repeat("0", 40)
	NULLPath  = "/dev/null"
)

type DiffTarget struct {
	Mode    string
	Path    string
	ObjId   string
	Content string
}

func CreateTargetFromHead(path string, repo *Repository, s *Status) (*DiffTarget, error) {
	e, ok := s.HeadTree[path]

	if !ok {
		return nil, data.ErrorEntriesNotExists
	}

	d, err := CreateTargetFromEntry(path, repo, e)

	if err != nil {
		return nil, err
	}

	return d, nil
}

func CreateTargetFromIndex(path string, repo *Repository) (*DiffTarget, error) {
	return RunCreateTargetFromIndex(0, path, repo)
}

func CreateTargetFromIndexWithStage(stage int, path string, repo *Repository) (*DiffTarget, error) {
	return RunCreateTargetFromIndex(stage, path, repo)
}

func RunCreateTargetFromIndex(stage int, path string, repo *Repository) (*DiffTarget, error) {
	e, ok := repo.i.EntryForPathWithStage(path, stage)

	if !ok {
		return nil, data.ErrorEntriesNotExists
	}

	d, err := CreateTargetFromEntry(path, repo, e)

	if err != nil {
		return nil, err
	}

	return d, nil
}

func CreateTargetFromEntry(path string, repo *Repository, e *con.Entry) (*DiffTarget, error) {

	if e == nil {
		return CreateTargetFromNothing(path)
	}

	o, err := repo.d.ReadObject(e.ObjId)
	if err != nil {
		return nil, err
	}

	blob, ok := o.(*con.Blob)

	if !ok {
		return nil, ErrorObjeToEntryConvError
	}

	return &DiffTarget{
		Path:    path,
		ObjId:   e.ObjId,
		Mode:    con.ModeToString(e.Mode),
		Content: blob.Content,
	}, nil
}

func CreateTargetFromFile(path string, s *Status, repo *Repository) (*DiffTarget, error) {
	content, err := repo.w.ReadFile(path)

	if err != nil {
		return nil, err
	}

	stat, ok := s.Stats[path]
	if !ok {
		return nil, data.ErrorEntriesNotExists
	}

	blob := &con.Blob{
		Content: content,
	}
	headerCon := data.GetStoreHeaderContent(blob)
	objId := crypt.HexDigestBySha1(headerCon)
	mode := con.ModeToString(data.ModeForStat(stat))

	return &DiffTarget{
		Path:    path,
		ObjId:   objId,
		Mode:    mode,
		Content: content,
	}, nil
}

func CreateTargetFromNothing(path string) (*DiffTarget, error) {
	return &DiffTarget{
		Path:  path,
		ObjId: NULLObjId,
	}, nil
}

func PrintConflictDiff(stage int, path string, s *Status, repo *Repository, w io.Writer) error {
	w.Write([]byte(fmt.Sprintf("* Unmerged path %s\n", path)))
	target1, err := CreateTargetFromIndexWithStage(stage, path, repo)
	if err != nil {
		return err
	}

	target2, err := CreateTargetFromFile(path, s, repo)
	if err != nil {
		return err
	}

	return PrintDiff(target1, target2, repo, w)

}

func DiffIndexWorkSpace(stage int, s *Status, repo *Repository, w io.Writer) error {

	var keys []string

	keys = append(keys, util.SortedKeys(s.WorkSpaceChanges)...)
	keys = append(keys, util.SortedKeys(s.Conflicts)...)

	for _, path := range keys {

		_, existsInConflict := s.Conflicts[path]
		if existsInConflict {
			PrintConflictDiff(stage, path, s, repo, w)
			return nil
		}

		status, ok := s.WorkSpaceChanges[path]

		if !ok {
			return ErrorInvalidChanges
		}

		switch status {
		case WORKSPACE_MODIFIED:
			{
				a, err := CreateTargetFromIndex(path, repo)
				if err != nil {
					return err
				}
				b, err := CreateTargetFromFile(path, s, repo)
				if err != nil {
					return err
				}
				err = PrintDiff(
					a,
					b,
					repo,
					w,
				)
				if err != nil {
					return err
				}

			}
		case WORKSPACE_DELETE:
			{
				a, err := CreateTargetFromIndex(path, repo)
				if err != nil {
					return err
				}
				b, err := CreateTargetFromNothing(path)
				if err != nil {
					return err
				}
				err = PrintDiff(
					a,
					b,
					repo,
					w,
				)
				if err != nil {
					return err
				}
			}
		}

	}

	return nil
}

func DiffHeadIndex(s *Status, repo *Repository, w io.Writer) error {

	for _, path := range util.SortedKeys(s.IndexChanges) {
		status, ok := s.IndexChanges[path]

		if !ok {
			return ErrorInvalidChanges
		}

		switch status {
		case INDEX_ADDED:
			{
				a, err := CreateTargetFromNothing(path)
				if err != nil {
					return err
				}
				b, err := CreateTargetFromIndex(path, repo)
				if err != nil {
					return err
				}
				err = PrintDiff(
					a,
					b,
					repo,
					w,
				)
				if err != nil {
					return err
				}

			}
		case INDEX_MODIFIED:
			{
				a, err := CreateTargetFromHead(path, repo, s)
				if err != nil {
					return err
				}
				b, err := CreateTargetFromIndex(path, repo)
				if err != nil {
					return err
				}
				err = PrintDiff(
					a,
					b,
					repo,
					w,
				)
				if err != nil {
					return err
				}

			}
		case INDEX_DELETE:
			{
				a, err := CreateTargetFromHead(path, repo, s)
				if err != nil {
					return err
				}
				b, err := CreateTargetFromNothing(path)
				if err != nil {
					return err
				}
				err = PrintDiff(
					a,
					b,
					repo,
					w,
				)
				if err != nil {
					return err
				}

			}
		}
	}

	return nil
}

type Differ interface {
	GetTreeDiffChange(oldObjId, newObjId string) map[string][]*con.Entry
}

func PrintCommitDiff(aObjId, bObjId string, repo *Repository, differ Differ, w io.Writer) error {
	diff := differ.GetTreeDiffChange(aObjId, bObjId)

	ss := util.SortedKeys(diff)

	for _, path := range ss {
		oldEntry := diff[path][0]
		newEntry := diff[path][1]
		aTarget, err := CreateTargetFromEntry(path, repo, oldEntry)
		if err != nil {
			return err
		}
		bTarget, err := CreateTargetFromEntry(path, repo, newEntry)
		if err != nil {
			return err
		}
		PrintDiff(
			aTarget,
			bTarget,
			repo,
			w)

	}

	return nil
}

func PrintDiff(a, b *DiffTarget, repo *Repository, w io.Writer) error {
	if a.Mode == b.Mode && a.ObjId == b.ObjId {
		return nil
	}

	a.Path = filepath.Join("a", a.Path)
	b.Path = filepath.Join("b", b.Path)

	w.Write([]byte(fmt.Sprintf("diff --git %s %s\n", a.Path, b.Path)))

	err := PrintDiffMode(a, b, w)

	if err != nil {
		return err
	}
	err = PrintDiffContent(a, b, repo, w)
	if err != nil {
		return err
	}

	return nil

}

func PrintDiffMode(a, b *DiffTarget, w io.Writer) error {
	if a.Mode == "" {
		w.Write([]byte(fmt.Sprintf("new file mode %s\n", b.Mode)))
	} else if b.Mode == "" {
		w.Write([]byte(fmt.Sprintf("deleted file mode %s\n", a.Mode)))
	} else if a.Mode != b.Mode {
		w.Write([]byte(fmt.Sprintf("old mode %s\n", a.Mode)))
		w.Write([]byte(fmt.Sprintf("new mode %s\n", b.Mode)))
	}

	return nil
}

func PrintDiffContent(a, b *DiffTarget, repo *Repository, w io.Writer) error {
	if a.ObjId == b.ObjId {
		return nil
	}

	fn := func() string {
		str := fmt.Sprintf("index %s..%s", ShortOid(a.ObjId, repo.d), ShortOid(b.ObjId, repo.d))

		if a.Mode == b.Mode {
			str += fmt.Sprintf(" %s\n", a.Mode)
		} else {
			str += "\n"
		}

		return str
	}

	w.Write([]byte(fn()))

	//?????????--- +++???????????????--- a.diffPath b.diffPath

	edits := myers.ComputeEdits(span.URI(a.Path), a.Content, b.Content)
	diff := fmt.Sprint(gotextdiff.ToUnified(a.Path, b.Path, a.Content, edits))

	w.Write([]byte(diff))
	return nil

}

func ShortOid(objId string, d *data.Database) string {
	return d.ShortObjId(objId)
}
