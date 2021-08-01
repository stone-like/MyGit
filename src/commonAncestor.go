package src

import (
	"errors"
	data "mygit/src/database"
	con "mygit/src/database/content"
	dataUtil "mygit/src/database/util"
	"mygit/util"
	"reflect"
)

var (
	PARENT_ONE = ":parentOne"
	PARENT_TWO = ":parentTwo"
	STALE      = ":stale"
	RESULT     = ":result"
)

var BothParentState = map[string]struct{}{
	PARENT_ONE: {},
	PARENT_TWO: {},
}

type CommonAncestors struct {
	queue   *util.PriorityQueue
	d       *data.Database
	flags   map[string]map[string]struct{}
	results *util.PriorityQueue
}

func GenerateCAS(firstObjId string, secondObjIds []string, d *data.Database) (*CommonAncestors, error) {

	cas := &CommonAncestors{
		queue:   util.GeneratePriorityQueue(),
		d:       d,
		flags:   make(map[string]map[string]struct{}),
		results: util.GeneratePriorityQueue(),
	}

	o1, err := d.ReadObject(firstObjId)

	if err != nil {
		return nil, err
	}

	c1, ok := o1.(*con.CommitFromMem)

	if !ok {
		return nil, ErrorObjeToEntryConvError
	}

	cas.queue.Push(&util.Item{
		Value:    c1,
		Priority: c1.Author.GetUnixTimeInt(),
	})

	cas.flags[firstObjId] = make(map[string]struct{})
	cas.flags[firstObjId][PARENT_ONE] = struct{}{}

	for _, secondObjId := range secondObjIds {
		o2, err := d.ReadObject(secondObjId)

		if err != nil {
			return nil, err
		}

		c2, ok := o2.(*con.CommitFromMem)

		if !ok {
			return nil, ErrorObjeToEntryConvError
		}

		cas.queue.Push(&util.Item{
			Value:    c2,
			Priority: c2.Author.GetUnixTimeInt(),
		})

		cas.flags[secondObjId] = make(map[string]struct{})
		cas.flags[secondObjId][PARENT_TWO] = struct{}{}
	}

	return cas, nil
}

func (cas *CommonAncestors) AllStale() (bool, error) {
	for _, i := range cas.queue.Queue {
		in := i.GetValue()
		c, ok := in.(*con.CommitFromMem)

		if !ok {
			return false, ErrorObjeToEntryConvError
		}

		//一つでもStaleでマークされていないやつがあればfalse
		if !cas.IsMarked(c.ObjId, STALE) {
			return false, nil
		}
	}

	return true, nil
}

func (cas *CommonAncestors) IsMarked(objId, flag string) bool {
	_, ok := cas.flags[objId]

	if !ok {
		return false
	}

	return util.HasKey(cas.flags[objId], flag)
}

func (cas *CommonAncestors) ProcessQueue() error {
	in := cas.queue.Pop()
	c, ok := in.(*con.CommitFromMem)
	if !ok {
		return ErrorObjeToEntryConvError
	}

	flags, ok := cas.flags[c.ObjId]

	if !ok {
		return nil
	}

	tempChildFlags := make(map[string]struct{})

	util.Copy(tempChildFlags, flags)

	if reflect.DeepEqual(flags, BothParentState) {
		//BothParentStateとイコールなので、一回resultフラグが立った奴はif側には来ない
		//なのでresultQueueに重複して追加されることもない
		//c is CommonAncestors
		//CASの場合はresultフラグを立ててBestCommonAncestorの候補とする
		cas.flags[c.ObjId][RESULT] = struct{}{}
		cas.results.Push(&util.Item{
			Value:    c,
			Priority: c.Author.GetUnixTimeInt(),
		})

		//ここより先の親はCASの親なので絶対にbestCommonAncestorにはなりえないのでStaleフラグ
		//STALEフラグはChildの方に立てたくないのでtempの方にSTALEを追加して渡す
		//ついでに上のRESULTフラグも引き継がせたくないし
		tempChildFlags[STALE] = struct{}{}
		err := cas.AddParents(c, tempChildFlags)

		if err != nil {
			return err
		}
	} else {
		//CASの親でなければStaleフラグはいらない
		err := cas.AddParents(c, tempChildFlags)

		if err != nil {
			return err
		}
	}

	return nil

}

func (cas *CommonAncestors) MakeResult() error {
	for {
		if cas.queue.Queue.Len() == 0 {
			break
		}

		allStaled, err := cas.AllStale()
		if err != nil {
			return err
		}

		if allStaled {
			break
		}

		err = cas.ProcessQueue()
		if err != nil {
			return err
		}
	}

	return nil
}

var ErrorInvalidCasState = errors.New("more than 1 Best common Ancestor occured or BestcommonAncestor not exists")

func (cas *CommonAncestors) FindCas() ([]string, error) {

	err := cas.MakeResult()
	if err != nil {
		return nil, err
	}

	var result []string

	//staleがついていないやつがBCA
	for _, i := range cas.results.Queue {
		in := i.GetValue()
		c, ok := in.(*con.CommitFromMem)

		if !ok {
			return nil, ErrorObjeToEntryConvError
		}

		if !cas.IsMarked(c.ObjId, STALE) {
			result = append(result, c.ObjId)
		}
	}
	return result, nil

}

func (cas *CommonAncestors) AddParents(c *con.CommitFromMem, childflags map[string]struct{}) error {
	if len(c.Parents) == 0 {
		return nil
	}

	for _, pObjId := range c.Parents {

		_, ok := cas.flags[pObjId]

		if !ok {
			cas.flags[pObjId] = make(map[string]struct{})
		}

		if util.IsContainOtherSet(cas.flags[pObjId], childflags) {
			//もうすでにparentがchildを含有しているなら
			continue
		}
		for f, _ := range childflags {

			cas.flags[pObjId][f] = struct{}{}
		}

		o, err := cas.d.ReadObject(pObjId)

		if err != nil {
			return err
		}

		pc, ok := o.(*con.CommitFromMem)

		if !ok {
			return ErrorObjeToEntryConvError
		}

		cas.queue.Push(&util.Item{
			Value:    pc,
			Priority: pc.Author.GetUnixTimeInt(),
		})

	}

	return nil
}

type RedunduntCas struct {
	redundant []string
}

func FilterCommit(commits []string, objId string, r *RedunduntCas, d *data.Database) error {
	if dataUtil.Contains(r.redundant, objId) {
		return nil
	}

	willRemove := append(r.redundant, objId)

	others := dataUtil.RemovedSlice(commits, willRemove)

	cas, err := GenerateCAS(objId, others, d)

	if err != nil {
		return err
	}

	_, err = cas.FindCas()
	if err != nil {
		return err
	}

	if cas.IsMarked(objId, PARENT_TWO) {
		r.redundant = append(r.redundant, objId)
	}

	for _, otherobjId := range others {
		if cas.IsMarked(otherobjId, PARENT_ONE) {
			r.redundant = append(r.redundant, otherobjId)
		}
	}

	return nil

}

var ErrorBCANotFound = errors.New("BCA not Found")

func GetBCA(headObjId, mergeObjId string, d *data.Database) (string, error) {
	//BCAが一つもなかった時ってエラーでよさそうだけど...
	commits, err := FindBCA(headObjId, mergeObjId, d)
	if err != nil {
		return "", err
	}
	if len(commits) == 0 {
		return "", ErrorBCANotFound
	}

	return commits[0], nil
}

func FindBCA(headObjId, mergeObjId string, d *data.Database) ([]string, error) {
	cas, err := GenerateCAS(headObjId, []string{mergeObjId}, d)
	if err != nil {
		return nil, err
	}
	commits, err := cas.FindCas()
	if err != nil {
		return nil, err
	}

	if len(commits) <= 1 {
		//commitsが1以下の時はBCAが見つかったか、一つもなかった時
		return commits, nil
	}

	r := &RedunduntCas{}

	//commitsが2以上ある場合
	for _, objId := range commits {
		err := FilterCommit(commits, objId, r, d)
		if err != nil {
			return nil, err
		}
	}

	return dataUtil.RemovedSlice(commits, r.redundant), nil

}
