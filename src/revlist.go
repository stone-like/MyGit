package src

import (
	con "mygit/src/database/content"
	dUtil "mygit/src/database/util"
	"mygit/util"
)

//別にすでに見たかどうかだけならflagsはmap[string]boolでいいんだけどブランチのExcludeとか実装するとflagが複数種類必要だから

//SEENはQUEUEに自分を追加したとき
//ADDEDはQUEUEに親を追加したとき
var (
	SEEN         = ":seen"
	ADDED        = ":added"
	UNINTERESTED = ":uninterested"
	TREE_SAME    = ":treesame"
)

type RevList struct {
	Limited bool
	repo    *Repository
	commits map[string]*con.CommitFromMem
	flags   map[string]map[string]struct{} //objIdごとにflagsがある、logではすでに見たコミットは重複して表示したくないのでそのフラグ
	queue   *util.PriorityQueue            //priorityQueue、コミットをコミットの時間順に並べる
	output  []*con.CommitFromMem           //uninterestingは入れないやつ
	prune   []string                       // log filepathの時使う
	diffs   map[string]*TreeDiff           // --patchの時に利用(現在patch実装していないのでいらないけど後々使う)
	filter  *PathFilter
	walk    bool
}

func (r *RevList) GetQueue() *util.PriorityQueue {
	return r.queue
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

		//logで普通のファイル名が来た時で、変化なしなら表示させる意味ないので
		if r.IsMarked(c.ObjId, TREE_SAME) {
			continue
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

func (r *RevList) RunGetAllCommits() ([]*con.CommitFromMem, error) {
	var commitList []*con.CommitFromMem

	for {

		if r.queue.Queue.Len() == 0 {
			break
		}

		v := r.queue.Pop()
		c, ok := v.(*con.CommitFromMem)
		if !ok {
			return nil, ErrorObjeToEntryConvError
		}

		//limitedの時はLimitQueueの時AddParentをやったからいいけどLimitedじゃないときはここでやる
		err := r.AddParent(c)
		if err != nil {
			return nil, err
		}

		if r.IsMarked(c.ObjId, UNINTERESTED) {
			continue
		}

		if r.IsMarked(c.ObjId, TREE_SAME) {
			continue
		}

		commitList = append(commitList, c)

	}

	return commitList, nil

}

func (r *RevList) GetAllCommitsOnLimit() ([]*con.CommitFromMem, error) {
	r.LimitQueue()
	return r.RunGetAllCommits()
}

func (r *RevList) GetAllCommits() ([]*con.CommitFromMem, error) {

	if r.Limited {
		return r.GetAllCommitsOnLimit()
	}

	return r.RunGetAllCommits()

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

		if !r.Limited {
			//limitedの時はLimitQueueの時AddParentをやったからいいけどLimitedじゃないときはここでやる
			err := r.AddParent(c)
			if err != nil {
				return err
			}
		}

		if r.IsMarked(c.ObjId, UNINTERESTED) {
			continue
		}

		if r.IsMarked(c.ObjId, TREE_SAME) {
			continue
		}
		//showPatchの場合はHeadだけでいい、Headのparentまでforで回す必要ない
		err := show(c)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *RevList) AddParent(c *con.CommitFromMem) error {
	if !r.walk || r.IsMarked(c.ObjId, ADDED) {
		return nil
	} else {
		r.Mark(c.ObjId, ADDED)

		if len(c.Parents) != 0 {

			var parents []*con.CommitFromMem
			if r.IsMarked(c.ObjId, UNINTERESTED) {

				for _, p := range c.Parents {
					o, err := r.repo.d.ReadObject(p)

					if err != nil {
						return err
					}

					//objIdからコミットまで持ってくるのは時間の情報が欲しいから
					parent, ok := o.(*con.CommitFromMem)

					if !ok {
						return ErrorObjeToEntryConvError
					}

					r.MarkParentUninteresting(parent)
					parents = append(parents, parent)
				}
			} else {
				//simplyCommitでもfileだけではなく普通のbranchを扱うことに注意
				//simplyCommitでTreeSameとなった親のみ表示
				// A -> B  ->  D [master]
				//    \   /
				//      C   [topic]の例でいえば、hello.txtはAでtest,Bでchanged,Dで変わらずchangedとなっているとして、
				//DからBのみTreeSameなのでBのルートのみをとり、かつD自体はTreeSameなのでOutput時にスルー
				//B->A、A->nilはtreeDiffがあるのでそれをoutput

				simpleParents, err := r.SimplifyCommit(c)
				if err != nil {
					return err
				}

				for _, p := range simpleParents {
					o, err := r.repo.d.ReadObject(p)

					if err != nil {
						return err
					}

					//objIdからコミットまで持ってくるのは時間の情報が欲しいから
					parent, ok := o.(*con.CommitFromMem)

					if !ok {
						return ErrorObjeToEntryConvError
					}
					parents = append(parents, parent)
				}
			}

			for _, p := range parents {
				r.EnqueueCommit(p)
			}
		}

		return nil

	}
}

func (r *RevList) SimplifyCommit(c *con.CommitFromMem) ([]string, error) {
	if len(r.prune) == 0 {
		return c.Parents, nil
	}

	var parents []string

	if len(c.Parents) != 0 {
		parents = c.Parents
	} else {
		parents = []string{""} //treeDiffで比べるためにParentが存在しないなら""と比較させる
	}

	for _, p := range parents {
		if !r.TreeDiffChanged(p, c.ObjId) {
			r.Mark(c.ObjId, TREE_SAME)

			return []string{p}, nil
		}
	}

	return c.Parents, nil

}

func (r *RevList) GetTreeDiffChange(oldObjId, newObjId string) map[string][]*con.Entry {
	td := GenerateTreeDiff(r.repo)
	td.CompareObjIdWithFilter(oldObjId, newObjId, r.filter)

	return td.Changes
}

func (r *RevList) TreeDiffChanged(oldObjId, newObjId string) bool {
	return len(r.GetTreeDiffChange(oldObjId, newObjId)) != 0
}

func GenerateRevList(repo *Repository, branches []string) (*RevList, error) {
	return RunGenerateRevList(true, repo, branches)
}

func GenerateRevListWithWalk(walk bool, repo *Repository, branches []string) (*RevList, error) {
	return RunGenerateRevList(walk, repo, branches)
}

//...や^でない場合はAddParentしない、これはcherrypick A BとしたときにA,Bしかいらなくて、A,Bの親はいらない
func RunGenerateRevList(walk bool, repo *Repository, branches []string) (*RevList, error) {
	if len(branches) == 0 {
		branches = []string{"HEAD"}
	}

	r := &RevList{
		repo:    repo,
		commits: make(map[string]*con.CommitFromMem),
		flags:   make(map[string]map[string]struct{}), //nestedMapの場合内側がmakeできていないので注意
		queue:   util.GeneratePriorityQueue(),
		walk:    walk,
	}
	//Generateの時点で時間順にCommitを並べる
	for _, b := range branches {
		err := r.HandleRevision(b)
		if err != nil {
			return nil, err
		}
	}

	//この時点でqueueがemptyということはfileなのでHEADを対象にhandleRevisionしてあげる

	if r.queue.Queue.Len() == 0 {
		err := r.HandleRevision("HEAD")
		if err != nil {
			return nil, err
		}
	}

	//PathFilterを作る
	r.filter = GeneratePathFilterWithTrie(
		util.GenerateTrieFromPaths(r.prune),
	)

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

	tempSlice := make([]string, 0, len(c.Parents))

	tempSlice = append(tempSlice, c.Parents...)

	for {

		if len(tempSlice) == 0 {
			break
		}

		objId := tempSlice[0]
		tempSlice = tempSlice[1:]

		if r.IsMarked(objId, UNINTERESTED) {
			break
		}

		r.Mark(objId, UNINTERESTED)

		pc, ok := r.commits[objId]

		if ok {
			tempSlice = append(tempSlice, pc.Parents...)
		}
	}
}

//walk=trueならAddParent
func (r *RevList) HandleRevision(name string) error {

	rangeSlice := dUtil.CheckRegExpSubString(RANGE, name)
	excludeSlice := dUtil.CheckRegExpSubString(EXCLUDE, name)

	stat, _ := r.repo.w.StatFile(name)
	if stat != nil {
		//branchNameじゃなくてFilePathだった場合
		r.prune = append(r.prune, name)
	} else if len(rangeSlice) != 0 {
		r.SetStartPoint(rangeSlice[0][1], true)
		r.SetStartPoint(rangeSlice[0][2], false)
		r.walk = true
	} else if len(excludeSlice) != 0 {
		r.SetStartPoint(excludeSlice[0][1], true)
		r.walk = true
	} else {
		//excludeに当てはまらない普通のやつ
		r.SetStartPoint(name, false)
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
