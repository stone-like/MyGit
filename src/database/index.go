package database

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"mygit/src/crypt"
	con "mygit/src/database/content"
	"mygit/src/database/util"
	"os"
	"sort"
)

var (
	REGULAR_MODE    = 0100644
	EXECUTABLE_MODE = 0100755
	MAX_PATH_SIZE   = 0xfff
	SIGNATURE       = "DIRC"
	VERSION         = uint32(2)
	HeaderBinLen    = 8
)

type HeaderBin struct {
	Version       uint32
	EntriesNumber uint32
}

var ErrorEntriesNotExists = errors.New("invalid path for Entries")

type Index struct {
	Path    string
	Entries map[string]con.Object
	Keys    []string
	Changed bool
	Parents map[string][]string
}

func GenerateIndex(path string) *Index {
	return &Index{
		Path:    path,
		Entries: make(map[string]con.Object),
		Parents: make(map[string][]string),
	}
}

func (i *Index) EntryForPath(path string) (*con.Entry, bool) {
	o, ok := i.Entries[path]

	if !ok {
		return nil, false
	}

	e, ok := o.(*con.Entry)
	if !ok {
		return nil, false
	}

	return e, true

}

func (i *Index) IsIndexed(path string) bool {
	_, entOk := i.Entries[path]
	_, parOk := i.Parents[path]

	return entOk || parOk
}
func (i *Index) IsIndexedFile(path string) bool {
	_, entOk := i.Entries[path]

	return entOk
}

type CreateFn func(path, objId string, state con.FileState) *con.Entry

var ErrorObjeToEntryConvError = errors.New("conversion error object to entry")

func (i *Index) GetEntries() ([]*con.Entry, error) {

	es := make([]*con.Entry, len(i.Keys))
	for ind, k := range i.Keys {
		e, ok := i.Entries[k].(*con.Entry)

		if !ok {
			return nil, ErrorObjeToEntryConvError
		}
		es[ind] = e
	}

	return es, nil
}

func (i *Index) RemoveEntry(path string) {
	o, ok := i.Entries[path]
	if !ok {
		return
	}

	e, ok := o.(*con.Entry)
	if !ok {
		return
	}

	newKeys := util.DeleteSpecificKey(i.Keys, path)

	i.Keys = newKeys
	newMap := util.DeleteFromMap(i.Entries, path)

	i.Entries = newMap

	//例えばpath=nested/bob.txtだとして、Parents[nested] -> nested/bob.txtがあるとき
	//Parents[nested]からnested/bob.txtを削除する

	for _, p := range e.ParentDirs(e.Path) {
		newParent := util.DeleteSpecificKey(i.Parents[p], e.Path)
		i.Parents[p] = newParent
		if len(i.Parents[p]) == 0 {
			newParnets := util.DeleteFromParnet(i.Parents, p)
			i.Parents = newParnets
		}
	}

}

func (i *Index) RemoveChildren(e *con.Entry) {
	//removeChildrenはparentsのvaluesを削除する
	//例として、nested/bob.txtがあったとして、nestedをaddしたとき、
	//Parentにはnested -> nested/bob.txtがある
	//path=nestedでParentsをみると、そのchildrenを削除
	children, ok := i.Parents[e.Path]

	if ok {
		for _, c := range children {
			i.RemoveEntry(c)
		}
	}
}

func (i *Index) DiscardConflicts(e *con.Entry) error {
	for _, p := range e.ParentDirs(e.Path) {
		i.RemoveEntry(p) //dummy.txt -> dummy.txt/nested.txtのときに対応
	}
	i.RemoveChildren(e) //dummy.txt/nested.txt -> dummy.txtの時に対応

	return nil
}

func (i *Index) Add(path, objId string, stat con.FileState, createIndexEntry CreateFn) error {

	e := i.CreateIndexEntry(path, objId, stat, createIndexEntry)
	err := i.DiscardConflicts(e)

	if err != nil {
		return err
	}

	i.StoreEntry(e, path)
	i.Changed = true

	return nil

}

func (i *Index) StoreParent(e *con.Entry) {

	path := e.Path
	for _, c := range e.ParentDirs(path) {

		children, ok := i.Parents[c]

		if ok {

			newChildren := append(children, path)
			i.Parents[c] = newChildren

		} else {
			i.Parents[c] = append([]string{}, path)
		}

	}
}

func (i *Index) StoreEntry(e *con.Entry, path string) {
	//pathが同じだったらcreateではなくupdateしたいはず

	if util.Contains(i.Keys, path) {
		i.Entries[path] = e
	} else {
		i.Keys = append(i.Keys, path)
		i.Entries[path] = e
	}

	i.StoreParent(e)

	// i.Keys = append(i.Keys, path)
	// i.Entries[path] = e

}

func IsExec(mode uint32) bool {
	return mode&0111 != 0
}

func (i *Index) CreateIndexEntry(path, objId string, stat con.FileState, createIndexEntry CreateFn) *con.Entry {
	return createIndexEntry(path, objId, stat)
}

func (i *Index) Remove(path string) {

	o, ok := i.Entries[path]
	if !ok {
		return
	}

	e, ok := o.(*con.Entry)
	if !ok {
		return
	}

	i.RemoveEntry(path)
	i.RemoveChildren(e)
	i.Changed = true
}

func (i *Index) Write(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	// l := lock.NewFileLock(path)
	// l.Lock()
	err = i.WriteContent(f, path)
	if err != nil {
		return err
	}
	// defer l.Unlock()

	return nil
}

func (i *Index) WriteContent(f *os.File, path string) error {

	if !i.Changed {
		return nil
	}

	var tempStr string
	tempStr += SIGNATURE

	hb := &HeaderBin{
		Version:       uint32(VERSION),
		EntriesNumber: uint32(len(i.Entries)),
	}

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, hb)

	tempStr += buf.String()

	sort.Strings(i.Keys)

	sort.Slice(i.Keys, func(j, k int) bool {
		return len(i.Keys[j]) < len(i.Keys[k])
	})

	for _, k := range i.Keys {
		tempStr += i.Entries[k].ToString()
	}

	content := crypt.DigestBySha1(tempStr)

	tempStr += content

	f.Write([]byte(tempStr))

	i.Changed = false

	return nil
}

func (i *Index) Load() error {
	_, err := os.Stat(i.Path)
	if err != nil {
		//.git/indexがない場合は何もしない
		return nil
	}

	// l := lock.NewFileLock(i.Path)
	// l.Lock()
	// defer l.Unlock()
	b, err := ioutil.ReadFile(i.Path)

	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(b)

	checkSum := &CheckSum{
		reader: buf,
	}

	count, err := i.ReadHeader(buf, checkSum)

	if err != nil {
		return err
	}

	err = i.ReadEntries(buf, checkSum, int(count))

	if err != nil {
		return err
	}

	err = i.ReadCheckSum(buf, checkSum)
	if err != nil {
		return err
	}

	return nil

}

var ErrorInvalidSig = errors.New("invalid signature")
var ErrorInvalidVersion = errors.New("invalid version")
var ErrorInvalidCheckSum = errors.New("invalid checksum")

var ENTRY_MIN_SIZE = 64

var CHECKSUM_SIZE = 20

func (i *Index) ReadCheckSum(r io.Reader, cs *CheckSum) error {
	sum := make([]byte, CHECKSUM_SIZE)
	err := binary.Read(r, binary.BigEndian, &sum)
	if err != nil {
		return err
	}

	if string(sum) != cs.GenerateHash() {
		return ErrorInvalidCheckSum
	}
	return nil
}

func (i *Index) ReadEntries(r io.Reader, cs *CheckSum, count int) error {

	for ind := 0; ind < count; ind++ {

		bs, err := cs.Read(r, ENTRY_MIN_SIZE)
		if err != nil {
			return err
		}
		//64byteまで読んで最後が0じゃなければ8byteずつ読んで０かどうか確かめる
		for {
			if bs[len(bs)-1] != byte(0) {
				eightbs, err := cs.Read(r, 8)
				if err != nil {
					return err
				}
				bs = append(bs, eightbs...)
			} else {
				break
			}
		}

		e, path, err := i.ParseEntry(bs)
		if err != nil {
			return err
		}

		i.StoreEntry(e, path)

	}

	return nil
}

func (i *Index) ParseEntry(bs []byte) (*con.Entry, string, error) {

	em := &con.EntryFromMem{}
	buf := bytes.NewBuffer(bs[:62]) //flagsまでを読み取る、pathからは自分で何とかする
	err := binary.Read(buf, binary.BigEndian, em)

	if err != nil {
		return nil, "", err
	}

	pathbytes := bs[62:]
	for {
		if pathbytes[len(pathbytes)-1] != byte(0) {
			break
		} else {
			pathbytes = pathbytes[:len(pathbytes)-1]
		}
	}

	e := em.ConvertToEntity(string(pathbytes))

	return e, string(pathbytes), nil
}

func (i *Index) ReadHeader(r io.Reader, cs *CheckSum) (uint32, error) {

	bs, err := cs.Read(r, len([]byte(SIGNATURE)))
	if err != nil {
		return 0, err
	}

	if string(bs) != SIGNATURE {
		return 0, ErrorInvalidSig
	}

	bs, err = cs.Read(r, HeaderBinLen)
	bsReader := bytes.NewReader(bs)
	if err != nil {
		return 0, err
	}

	bin := &HeaderBin{}
	err = binary.Read(bsReader, binary.BigEndian, bin)
	if err != nil {
		return 0, err
	}

	if bin.Version != VERSION {
		return 0, ErrorInvalidVersion
	}

	return bin.EntriesNumber, nil

}

func (i *Index) StatMatch(e *con.Entry, stat con.FileState) bool {
	return e.Size == stat.Size() && e.Mode == ModeForStat(stat)
}

func ModeForStat(stat con.FileState) int {
	var mode int
	if IsExec(uint32(stat.Mode())) {
		mode = EXECUTABLE_MODE
	} else {
		mode = REGULAR_MODE
	}

	return mode
}

func (i *Index) UpdateEntryStat(e *con.Entry, stat con.FileState) {
	UpdateStat(e, stat)
	i.Changed = true
}
