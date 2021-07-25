package src

import (
	con "mygit/src/database/content"
	dUtil "mygit/src/database/util"
	"mygit/util"
)

//別にすでに見たかどうかだけならflagsはmap[string]boolでいいんだけどブランチのExcludeとか実装するとflagが複数種類必要だから

//SEENはQUEUEに自分を追加したとき
//ADDEDはQUEUEに親を追加したときとき
var (
	SEEN         = ":seen"
	ADDED        = ":added"
	UNINTERESTED = ":uninterested"
)

type RevList struct {
	Limited bool
	repo    *Repository
	commits map[string]*con.CommitFromMem
	flags   map[string]map[string]struct{} //objIdごとにflagsがある、logではすでに見たコミットは重複して表示したくないのでそのフラグ
	queue   *util.PriorityQueue            //priorityQueue、コミットをコミットの時間順に並べる
	output  []*con.CommitFromMem           //uninterestingは入れないやつ
}

// A -> B -> C
//        -> E -> F -> GでCのみにできるか？

//RevListでやりたいことは二つで、
//最初にGenerateでbranchesをqueueに入れる
//次に生成したqueueから時間順にコミットを取り出してその親をQueueに入れる
func (r *RevList) EachCommit(show func(c *con.CommitFromMem) error) error {

	//outputする前にuninterestingがらみでqueueをいろいろ整理する必要がある
	if r.Limited {
		r.LimitQueue()
	}
	err := r.OutputCommit(show)

	return err
}

func (r *RevList) HasInteresting() (bool, error) {

	if len(r.output) != 0 && r.queue.Queue.Len() != 0 {
		// A <- B   <- C <- D master
		//       <- E <- F <- G <- H <- J <- K topic
		// E <- C <- F  <- D　<- Gの時間順
		//上記においてtopic..masterをしたとき
		//このときCがEより先に呼ばれることでBまで到達できてしまう

		//ここでoutput_oldestとqueue_newestを比べている理由は、
		//uninterestinguninterestingからinterestingに到達できるか

		oldest_out := r.output[len(r.output)-1]
		i := r.queue.SeeFirst()
		newest_in, ok := i.(*con.CommitFromMem)

		if !ok {
			return false, ErrorObjeToEntryConvError
		}
		if oldest_out.Author.GetUnixTimeInt() <= newest_in.Author.GetUnixTimeInt() {
			return true, nil
		}
	}

	for _, in := range r.queue.Queue {
		c, ok := in.GetValue().(*con.CommitFromMem)
		if !ok {
			return false, ErrorObjeToEntryConvError
		}

		if !r.IsMarked(c.ObjId, UNINTERESTED) {
			return true, nil
		}
	}

	return false, nil
}

//いったんここで親のUninterestingまで見てしまう
func (r *RevList) LimitQueue() error {
	if r.queue.Queue.Len() == 0 {
		return nil
	}

	for {
		ok, err := r.HasInteresting()
		if err != nil {
			return err
		}

		if !ok {
			break
		}

		v := r.queue.Pop()
		c, ok := v.(*con.CommitFromMem)
		if !ok {
			return ErrorObjeToEntryConvError
		}
		err = r.AddParent(c)
		if err != nil {
			return err
		}

		//UninterestingじゃなければOutputに
		if !r.IsMarked(c.ObjId, UNINTERESTED) {
			r.output = append(r.output, c)
		}

	}

	//filterしたoutputを新しいqueueにする
	//新しいqueueを作る
	r.queue = util.GeneratePriorityQueue()
	for _, c := range r.output {
		//もうSEENになっているのでEnqueueではなくそのままQueueに入れる
		r.queue.Push(&util.Item{
			Value:    c,
			Priority: c.Author.GetUnixTimeInt(),
		})
	}

	return nil

}

func (r *RevList) OutputCommit(show func(c *con.CommitFromMem) error) error {
	for {

		if r.queue.Queue.Len() == 0 {
			break
		}

		v := r.queue.Pop()
		c, ok := v.(*con.CommitFromMem)
		if !ok {
			return ErrorObjeToEntryConvError
		}
		err := show(c)
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

			if r.IsMarked(c.ObjId, UNINTERESTED) {
				r.MarkParentUninteresting(parent)
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

var (
	RANGE   = `^(.*)\.\.(.*)$` //some..anyの形、someをexclude
	EXCLUDE = `^\^(.+)$`       //^someの形,someをexclude
	//excludeしたやつはparentCommitもExclude
)

func (r *RevList) SetStartPoint(branchName string, uninteresting bool) error {
	//interesting = trueなら普通に、falseならexclude
	if branchName == "" {
		branchName = "HEAD"
	}

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

	r.commits[c.ObjId] = c

	r.EnqueueCommit(c)

	//interestingじゃない時、exclude対象の時
	if uninteresting {
		r.Limited = true
		r.Mark(c.ObjId, UNINTERESTED)
		r.MarkParentUninteresting(c)
	}

	return nil

}

func (r *RevList) MarkParentUninteresting(c *con.CommitFromMem) {
	for {

		if c == nil || c.Parent == "" {
			break
		}

		if r.IsMarked(c.Parent, UNINTERESTED) {
			break
		}

		r.Mark(c.Parent, UNINTERESTED)

		pc, ok := r.commits[c.Parent]

		if ok {
			c = pc
		} else {
			c = nil
		}
	}
}

func (r *RevList) HandleRevision(branchName string) error {

	rangeSlice := dUtil.CheckRegExpSubString(RANGE, branchName)
	excludeSlice := dUtil.CheckRegExpSubString(EXCLUDE, branchName)

	if len(rangeSlice) != 0 {
		r.SetStartPoint(rangeSlice[0][1], true)
		r.SetStartPoint(rangeSlice[0][2], false)
	} else if len(excludeSlice) != 0 {
		r.SetStartPoint(excludeSlice[0][1], true)
	} else {
		//excludeに当てはまらない普通のやつ
		r.SetStartPoint(branchName, false)
	}

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
