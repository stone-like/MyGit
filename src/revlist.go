package src

import (
	con "mygit/src/database/content"
	"mygit/util"
)

//別にすでに見たかどうかだけならflagsはmap[string]boolでいいんだけどブランチのExcludeとか実装するとflagが複数種類必要だから

var (
	SEEN  = ":seen"
	ADDED = ":added"
)

type RevList struct {
	repo    *Repository
	commits map[string]*con.CommitFromMem
	flags   map[string]map[string]struct{} //objIdごとにflagsがある、logではすでに見たコミットは重複して表示したくないのでそのフラグ
	queue   *util.PriorityQueue            //priorityQueue、コミットをコミットの時間順に並べる
}

//RevListでやりたいことは二つで、
//最初にGenerateでbranchesをqueueに入れる
//次に生成したqueueから時間順にコミットを取り出してその親をQueueに入れる
func (r *RevList) EachCommit(show func(c *con.CommitFromMem) error) error {
	for {
		if r.queue.Queue.Len() == 0 {
			break
		}

		v := r.queue.Pop()
		c, ok := v.(*con.CommitFromMem)
		if !ok {
			return ErrorObjeToEntryConvError
		}

		err := r.AddParent(c)
		if err != nil {
			return err
		}

		err = show(c)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *RevList) AddParent(c *con.CommitFromMem) error {
	if r.IsMarked(c.ObjId, ADDED) {
		return nil
	} else {
		r.Mark(c.ObjId, ADDED)

		if c.Parent != "" {
			o, err := r.repo.d.ReadObject(c.Parent)

			if err != nil {
				return err
			}

			//objIdからコミットまで持ってくるのは時間の情報が欲しいから
			parent, ok := o.(*con.CommitFromMem)

			if !ok {
				return ErrorObjeToEntryConvError
			}

			r.EnqueueCommit(parent)
		}

		return nil

	}
}

func GenerateRevList(repo *Repository, branches []string) (*RevList, error) {
	if len(branches) == 0 {
		branches = []string{"HEAD"}
	}

	r := &RevList{
		repo:    repo,
		commits: make(map[string]*con.CommitFromMem),
		flags:   make(map[string]map[string]struct{}), //nestedMapの場合内側がmakeできていないので注意
		queue:   util.GeneratePriorityQueue(),
	}
	//Generateの時点で時間順にCommitを並べる
	for _, b := range branches {
		err := r.HandleRevision(b)
		if err != nil {
			return nil, err
		}
	}

	return r, nil

}

func (r *RevList) HandleRevision(branchName string) error {
	rev, err := ParseRev(branchName)
	if err != nil {
		return err
	}

	objId, err := ResolveRev(rev, r.repo)
	if err != nil {
		return err
	}

	o, err := r.repo.d.ReadObject(objId)

	if err != nil {
		return err
	}

	//objIdからコミットまで持ってくるのは時間の情報が欲しいから
	c, ok := o.(*con.CommitFromMem)

	if !ok {
		return ErrorObjeToEntryConvError
	}

	r.EnqueueCommit(c)

	return nil
}

func (r *RevList) IsMarked(objId, flag string) bool {
	m, objOk := r.flags[objId]
	_, flagOk := m[flag]
	return objOk && flagOk
}

func (r *RevList) Mark(objId, flag string) {
	_, objOk := r.flags[objId]
	if !objOk {
		//nestedMapの場合内側がmakeできていないので、ここで作る
		r.flags[objId] = make(map[string]struct{})
	}
	r.flags[objId][flag] = struct{}{}
}

func (r *RevList) EnqueueCommit(c *con.CommitFromMem) {
	if r.IsMarked(c.ObjId, SEEN) {
		return
	} else {
		r.queue.Push(
			&util.Item{
				Value:    c,
				Priority: c.Author.GetUnixTimeInt(),
			},
		)
		r.Mark(c.ObjId, SEEN)
	}
}
