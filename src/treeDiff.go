package src

import (
	con "mygit/src/database/content"
	"reflect"
)

type TreeDiff struct {
	Changes map[string][]*con.Entry
	repo    *Repository
}

func TreeDiffGenerateAndCompareCommit(fromObjId, toObjId string, repo *Repository) (*TreeDiff, error) {
	t := &TreeDiff{
		Changes: make(map[string][]*con.Entry),
		repo:    repo,
	}

	err := t.CompareObjId(fromObjId, toObjId)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func GenerateTreeDiff(repo *Repository) *TreeDiff {
	return &TreeDiff{
		Changes: make(map[string][]*con.Entry),
		repo:    repo,
	}
}

func (t *TreeDiff) CompareObjIdWithFilter(fromObjId, toObjId string, filter *PathFilter) error {
	return t.Compare(fromObjId, toObjId, filter)
}

func (t *TreeDiff) CompareObjId(fromObjId, toObjId string) error {
	return t.Compare(fromObjId, toObjId, GeneratePathFilter())
}

func (t *TreeDiff) Compare(fromObjId, toObjId string, filter *PathFilter) error {
	aObjId := fromObjId
	bObjId := toObjId
	if aObjId == bObjId {
		return nil
	}

	aTree, err := GetTree(aObjId, t.repo)
	if err != nil {
		return err
	}
	bTree, err := GetTree(bObjId, t.repo)
	if err != nil {
		return err
	}
	//btoObjIdが""の時、bTreeはnilなのでそのままEntriesを呼ぶとError

	var aEntries map[string]con.Object
	var bEntries map[string]con.Object
	if aTree != nil {
		aEntries = aTree.Entries
	}
	if bTree != nil {
		bEntries = bTree.Entries
	}

	err = t.DetectDeletions(aEntries, bEntries, filter)
	if err != nil {
		return err
	}
	err = t.DetectAddtions(aEntries, bEntries, filter)
	if err != nil {
		return err
	}

	return nil
}

func (t *TreeDiff) DetectDeletions(aEntries, bEntries map[string]con.Object, filter *PathFilter) error {
	//aEntriesにはあって、bEntriesにはないものをみつける

	fn := func(k string, v con.Object) error {
		path := k
		other, ok := bEntries[k]
		ev, evOk := v.(*con.Entry)
		if !evOk {
			return ErrorObjeToEntryConvError
		}

		subFilter := filter.Join(k)

		if !ok {
			var aObjId string
			if ev.IsTree() {
				aObjId = ev.GetObjId()
				err := t.CompareObjIdWithFilter(aObjId, "", subFilter)

				if err != nil {
					return err
				}
			} else {
				t.Changes[path] = []*con.Entry{ev, nil}
			}
		} else {
			eo, eoOk := other.(*con.Entry)
			if !eoOk {
				return ErrorObjeToEntryConvError
			}

			if reflect.DeepEqual(ev, eo) {
				return nil
			}

			var aObjId string
			var bObjId string

			if ev.IsTree() {
				aObjId = ev.GetObjId()
			}
			if eo.IsTree() {
				bObjId = eo.GetObjId()
			}

			err := t.CompareObjIdWithFilter(aObjId, bObjId, subFilter)

			if err != nil {
				return err
			}

			if !ev.IsTree() || !eo.IsTree() {

				targetV := ev
				targetO := eo
				if ev.IsTree() {
					targetV = nil
				}
				if eo.IsTree() {
					targetO = nil
				}
				t.Changes[path] = []*con.Entry{targetV, targetO}
			}

		}

		return nil
	}

	err := filter.EachEntry(aEntries, fn)

	if err != nil {
		return err
	}

	return nil

}

func (t *TreeDiff) DetectAddtions(aEntries, bEntries map[string]con.Object, filter *PathFilter) error {
	//bEntriesにはあって、aEntriesにはないものをみつける
	fn := func(k string, v con.Object) error {
		path := k
		_, ok := aEntries[k]
		ev, evOk := v.(*con.Entry)
		if !evOk {
			return ErrorObjeToEntryConvError
		}
		if ok {
			//aEntriesにあるものはスキップ
			return nil
		}

		subFilter := filter.Join(k)

		//ここから先はaEntriesにないもの
		if ev.IsTree() {
			err := t.CompareObjIdWithFilter("", ev.GetObjId(), subFilter)
			if err != nil {
				return err
			}
		} else {
			t.Changes[path] = []*con.Entry{nil, ev}
		}

		return nil
	}

	err := filter.EachEntry(bEntries, fn)

	if err != nil {
		return err
	}

	return nil
}

func GetTree(objId string, repo *Repository) (*con.Tree, error) {
	if objId == "" {
		return nil, nil
	}

	o, err := repo.d.ReadObject(objId)

	if err != nil {
		return nil, err
	}

	switch v := o.(type) {
	case *con.CommitFromMem:
		{
			o2, err := repo.d.ReadObject(v.Tree)
			if err != nil {
				return nil, err
			}

			t, ok := o2.(*con.Tree)
			if !ok {
				return nil, ErrorObjeToEntryConvError
			}

			return t, nil
		}
	case *con.Tree:
		{
			return v, nil
		}
	default:
		return nil, ErrorObjeToEntryConvError
	}
}
