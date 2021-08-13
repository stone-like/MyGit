package src

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	con "mygit/src/database/content"
	"mygit/src/database/lock"
	"mygit/src/database/util"
	ers "mygit/src/errors"
	"os"
	"path/filepath"
	"strings"
)

type Sequencer struct {
	repo    *Repository
	Path    string
	Command []*con.CommitFromMem
}

func GenerateSequencer(repo *Repository) *Sequencer {
	seqPath := filepath.Join(repo.r.Path, "sequencer")
	return &Sequencer{
		repo: repo,
		Path: seqPath,
	}
}

//todoは途中でconflictして再開したときにcommitする予定のcommitObjIdを書き込む
func (s *Sequencer) GetToDoPath() string {
	return filepath.Join(s.Path, "todo")
}

//abort-safetyはコミットが成功した最後のcommitObjIdを書く
func (s *Sequencer) GetAbortPath() string {
	return filepath.Join(s.Path, "abort-safety")
}

//headはcherryPickを始める前のcommitObjIdを書く
func (s *Sequencer) GetHeadPath() string {
	return filepath.Join(s.Path, "head")
}

func (s *Sequencer) IsExists(path string) bool {
	stat, _ := os.Stat(path)

	if stat == nil {
		return false
	}

	return true
}

func (s *Sequencer) IsDir(path string) bool {
	stat, _ := os.Stat(path)

	return stat.IsDir()
}

func (s *Sequencer) Write(path, content string) error {
	l := lock.NewFileLock(path)
	l.Lock()
	defer l.Unlock()

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	f.Write([]byte(content))
	f.Write([]byte("\n"))
	return nil

}

var UNSAFE_MESSAGE = "You seem to have moved HEAD. Not rewinding, check your HEAD!"

func (s *Sequencer) Abort() error {

	beforeCherryContent, err := ioutil.ReadFile(s.GetHeadPath())
	if err != nil {
		return err
	}
	beforeCherryObjId := strings.TrimSpace(string(beforeCherryContent))

	latestContent, err := ioutil.ReadFile(s.GetAbortPath())
	if err != nil {
		return err
	}
	latestObjId := strings.TrimSpace(string(latestContent))

	currentHeadObjId, err := s.repo.r.ReadHead()
	if err != nil {
		return err
	}

	err = s.Clear()
	if err != nil {
		return err
	}

	//abortの前にHEADを移動しちゃったとき
	if latestObjId != currentHeadObjId {
		return &ers.SequenceAbortError{
			Message: UNSAFE_MESSAGE,
		}
	}

	//resetHardでcherryPickする前まで戻す
	err = HanldeHard(beforeCherryObjId, s.repo)
	if err != nil {
		return err
	}

	//headも戻しておく
	origHead, err := s.repo.r.UpdateHead(beforeCherryObjId)
	if err != nil {
		return err
	}

	//ORIG_HEADも戻す
	err = s.repo.r.UpdateRef(s.repo.r.OrigHeadPath(), origHead)
	if err != nil {
		return err
	}

	return nil

}

func (s *Sequencer) Start() error {
	err := os.MkdirAll(s.Path, os.ModePerm)
	if err != nil {
		return err
	}

	if !s.IsDir(s.Path) {
		return ErrorInvalidSequqncerState
	}

	headObjId, err := s.repo.r.ReadHead()
	if err != nil {
		return err
	}
	err = s.Write(s.GetHeadPath(), headObjId)
	if err != nil {
		return err
	}
	err = s.Write(s.GetAbortPath(), headObjId)
	if err != nil {
		return err
	}

	return nil

}

func (s *Sequencer) Push(c *con.CommitFromMem) {
	s.Command = append(s.Command, c)
}

func (s *Sequencer) UpdateAbortSafetyLatest() error {
	//commitのpickが一回完了するごとに、abort-safetyはコミットが成功した最後のcommitObjIdを書くので、
	//abort-safetyを更新していく

	//RunPickにてエラーがないならHeadも更新されている
	curHeadObjId, err := s.repo.r.ReadHead()
	if err != nil {
		return err
	}

	return s.Write(s.GetAbortPath(), curHeadObjId)
}

func (s *Sequencer) Shift() (*con.CommitFromMem, error) {

	if len(s.Command) == 0 {
		return nil, nil
	}

	lastCommit := s.Command[0]
	s.Command = s.Command[1:]

	return lastCommit, nil
}

func (s *Sequencer) NextCommand() *con.CommitFromMem {

	if len(s.Command) == 0 {
		return nil
	}

	return s.Command[0]
}

//todoFileにはcherryPickの際にconflictしてmergeされなかったcommitが下記のように書き込まれる
// pick shortObjId message(firstLine)
// pick ...

//\Sは非空白文字のこと
var PickExp = `^pick (\S+) (.*)$`

func (s *Sequencer) ParseToDo(content string) error {
	buf := bytes.NewBuffer([]byte(content))

	sc := bufio.NewScanner(buf)

	for sc.Scan() {
		pickExped := util.CheckRegExpSubString(PickExp, sc.Text())
		if len(pickExped) == 0 {
			return ErrorInvalidToDoContent
		}

		pickObjId := pickExped[0][1]

		objId, err := PrefixMatch(pickObjId, s.repo)
		if err != nil {
			return err
		}

		o, err := s.repo.d.ReadObject(objId)

		c, ok := o.(*con.CommitFromMem)
		if !ok {
			return ErrorObjeToEntryConvError
		}

		s.Command = append(s.Command, c)
	}

	return nil
}

func (s *Sequencer) Load() error {
	if !s.IsDir(s.Path) {
		return ErrorInvalidSequqncerState
	}

	content, err := ioutil.ReadFile(s.GetToDoPath())
	if err != nil {
		return err
	}

	return s.ParseToDo(string(content))

}

func (s *Sequencer) Clear() error {
	return os.RemoveAll(s.Path)
}

func (s *Sequencer) WriteToDo() error {
	if !s.IsExists(s.GetToDoPath()) {
		return ErrorInvalidToDoFileState
	}

	f, err := os.Create(s.GetToDoPath())
	defer f.Close()
	if err != nil {
		return err
	}

	for _, c := range s.Command {
		shortObjId := s.repo.d.ShortObjId(c.ObjId)
		f.Write([]byte(fmt.Sprintf("pick %s %s\n", shortObjId, c.GetFirstLineMessage())))
	}

	return nil

}

var ErrorInvalidToDoContent = errors.New("todo content line must match target exp")

var ErrorInvalidToDoFileState = errors.New("todo require exist")

var ErrorInvalidSequqncerState = errors.New(".git/sequencer must be Dir")
