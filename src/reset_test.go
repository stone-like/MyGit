package src

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	data "mygit/src/database"
	con "mygit/src/database/content"

	"github.com/stretchr/testify/assert"
)

func PrepareTwoCommitForReset(t *testing.T) string {
	cur, err := os.Getwd()
	assert.NoError(t, err)
	tempPath, err := ioutil.TempDir(cur, "")
	assert.NoError(t, err)

	xxxPath := filepath.Join(tempPath, "xxx")
	err = os.MkdirAll(xxxPath, os.ModePerm)
	assert.NoError(t, err)

	helloPath := CreateFiles(t, tempPath, "hello.txt", "test\n")
	dummyPath := CreateFiles(t, xxxPath, "dummy.txt", "test2\n")

	is := []string{tempPath}
	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "test", &buf)
	assert.NoError(t, err)

	CreateFiles(t, xxxPath, "dummy2.txt", "test\n")
	f, err := os.Create(helloPath)
	assert.NoError(t, err)
	f.Write([]byte("helloChanged"))
	f.Close()
	os.RemoveAll(dummyPath)

	assert.NoError(t, err)
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "test2", &buf)
	assert.NoError(t, err)

	return tempPath //こうしているのはなぜかcleanUp関数を返そうとすると最後のコミットまでいかない
}

func TestParseArgsFirstToRev(t *testing.T) {
	tempPath := PrepareTwoCommitForReset(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	rev, err := ParseRev("@^")
	assert.NoError(t, err)
	objId, err := ResolveRev(rev, repo)
	assert.NoError(t, err)
	res := &Reset{
		Args: []string{"@^"},
		repo: repo,
	}

	err = res.SelectCommitObjId(res.Args)
	assert.NoError(t, err)

	assert.Equal(t, 0, len(res.Args))
	assert.Equal(t, objId, res.CommitObjId)
}

func TestParseArgsFirstToFileName(t *testing.T) {
	tempPath := PrepareTwoCommitForReset(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	rev, err := ParseRev("@")
	assert.NoError(t, err)
	objId, err := ResolveRev(rev, repo)
	assert.NoError(t, err)

	res := &Reset{
		Args: []string{"hello.txt"},
		repo: repo,
	}

	err = res.SelectCommitObjId(res.Args)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(res.Args))
	assert.Equal(t, objId, res.CommitObjId)
}

//resetでファイル単位とすべて
//ファイル単位の時はHEADがそのままなこと
//mixed,soft,hardそれぞれの挙動
func TestPerFileMixedOnly(t *testing.T) {
	tempPath := PrepareTwoCommitForReset(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	helloPath := filepath.Join(tempPath, "hello.txt")

	d, err := ioutil.ReadFile(helloPath)
	assert.NoError(t, err)
	assert.Equal(t, "helloChanged", string(d))

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	rev, err := ParseRev("@")
	assert.NoError(t, err)
	beforeResetHeadObjId, err := ResolveRev(rev, repo)
	assert.NoError(t, err)

	res := &Reset{Args: []string{"@^", "hello.txt"}, repo: repo, Option: &ResetOption{}}

	err = RunReset(res)
	assert.NoError(t, err)

	o := res.repo.i.Entries[data.EntryKey{Path: "hello.txt", Stage: 0}]
	o2, err := res.repo.d.ReadObject(o.GetObjId())
	assert.NoError(t, err)
	b := o2.(*con.Blob)
	//hello.txtが@^に戻っていること
	assert.Equal(t, b.Content, "test\n")

	//mixedなのでworkSpaceは変わらないこと
	d, err = ioutil.ReadFile(helloPath)
	assert.NoError(t, err)
	assert.Equal(t, "helloChanged", string(d))

	//headが変わってないこと
	headObjId, err := res.repo.r.ReadHead()
	assert.NoError(t, err)
	assert.Equal(t, beforeResetHeadObjId, headObjId)

}

func TestPerFileMixed(t *testing.T) {
	tempPath := PrepareTwoCommitForReset(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	helloPath := filepath.Join(tempPath, "hello.txt")

	d, err := ioutil.ReadFile(helloPath)
	assert.NoError(t, err)
	assert.Equal(t, "helloChanged", string(d))

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	rev, err := ParseRev("@")
	assert.NoError(t, err)
	beforeResetHeadObjId, err := ResolveRev(rev, repo)
	assert.NoError(t, err)

	res := &Reset{Args: []string{"@^"}, repo: repo, Option: &ResetOption{}}

	err = RunReset(res)
	assert.NoError(t, err)

	o := res.repo.i.Entries[data.EntryKey{Path: "hello.txt", Stage: 0}]
	o2, err := res.repo.d.ReadObject(o.GetObjId())
	assert.NoError(t, err)
	b := o2.(*con.Blob)
	//hello.txtが@^に戻っていること
	assert.Equal(t, b.Content, "test\n")

	//mixedなのでworkSpaceは変わらないこと
	d, err = ioutil.ReadFile(helloPath)
	assert.NoError(t, err)
	assert.Equal(t, "helloChanged", string(d))

	//indexにdummy.txtもあること
	o = res.repo.i.Entries[data.EntryKey{Path: "xxx/dummy.txt", Stage: 0}]
	o2, err = res.repo.d.ReadObject(o.GetObjId())
	assert.NoError(t, err)
	b = o2.(*con.Blob)

	assert.Equal(t, b.Content, "test2\n")

	//indexにdummy2.txtがないこと
	_, ok := res.repo.i.Entries[data.EntryKey{Path: "xxx/dummy2.txt", Stage: 0}]
	assert.False(t, ok)
	//workspaceが変わらないのでworkspaceにはdummy2.txtがあること
	stat, _ := repo.w.StatFile("xxx/dummy2.txt")
	assert.NotNil(t, stat)

	//headが変わっていること
	headObjId, err := res.repo.r.ReadHead()
	assert.NoError(t, err)
	assert.NotEqual(t, beforeResetHeadObjId, headObjId)

	//ORIG_HEADがreset前のHEADなこと
	oh, err := ioutil.ReadFile(filepath.Join(tempPath, ".git/ORIG_HEAD"))
	assert.NoError(t, err)
	str := strings.TrimSpace(string(oh))
	assert.Equal(t, beforeResetHeadObjId, str)
}

func TestResetHard(t *testing.T) {
	tempPath := PrepareTwoCommitForReset(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	helloPath := filepath.Join(tempPath, "hello.txt")

	d, err := ioutil.ReadFile(helloPath)
	assert.NoError(t, err)
	assert.Equal(t, "helloChanged", string(d))

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	rev, err := ParseRev("@")
	assert.NoError(t, err)
	beforeResetHeadObjId, err := ResolveRev(rev, repo)
	assert.NoError(t, err)

	res := &Reset{Args: []string{"@^"}, repo: repo, Option: &ResetOption{hasHard: true}}

	err = RunReset(res)
	assert.NoError(t, err)

	//hello.txtが戻っていること
	o := res.repo.i.Entries[data.EntryKey{Path: "hello.txt", Stage: 0}]
	o2, err := res.repo.d.ReadObject(o.GetObjId())
	assert.NoError(t, err)
	b := o2.(*con.Blob)

	assert.Equal(t, b.Content, "test\n")

	d, err = ioutil.ReadFile(helloPath)
	assert.NoError(t, err)
	assert.Equal(t, "test\n", string(d))

	//dummy.txtもあること
	o = res.repo.i.Entries[data.EntryKey{Path: "xxx/dummy.txt", Stage: 0}]
	o2, err = res.repo.d.ReadObject(o.GetObjId())
	assert.NoError(t, err)
	b = o2.(*con.Blob)

	assert.Equal(t, b.Content, "test2\n")

	d, err = ioutil.ReadFile(filepath.Join(tempPath, "xxx/dummy.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "test2\n", string(d))

	//全体が戻るのでdummy2が存在しないこと
	_, ok := res.repo.i.Entries[data.EntryKey{Path: "xxx/dummy2.txt", Stage: 0}]
	assert.False(t, ok)
	stat, _ := repo.w.StatFile("xxx/dummy2.txt")
	assert.Nil(t, stat)

	//headが変わっていること
	headObjId, err := res.repo.r.ReadHead()
	assert.NoError(t, err)
	assert.NotEqual(t, beforeResetHeadObjId, headObjId)

	//ORIG_HEADがreset前のHEADなこと
	oh, err := ioutil.ReadFile(filepath.Join(tempPath, ".git/ORIG_HEAD"))
	assert.NoError(t, err)
	str := strings.TrimSpace(string(oh))
	assert.Equal(t, beforeResetHeadObjId, str)

}

func TestResetSoft(t *testing.T) {
	tempPath := PrepareTwoCommitForReset(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	helloPath := filepath.Join(tempPath, "hello.txt")

	d, err := ioutil.ReadFile(helloPath)
	assert.NoError(t, err)
	assert.Equal(t, "helloChanged", string(d))

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	rev, err := ParseRev("@")
	assert.NoError(t, err)
	beforeResetHeadObjId, err := ResolveRev(rev, repo)
	assert.NoError(t, err)

	res := &Reset{Args: []string{"@^"}, repo: repo, Option: &ResetOption{hasSoft: true}}

	err = RunReset(res)
	assert.NoError(t, err)

	//hello.txtがそのままなこと
	o := res.repo.i.Entries[data.EntryKey{Path: "hello.txt", Stage: 0}]
	o2, err := res.repo.d.ReadObject(o.GetObjId())
	assert.NoError(t, err)
	b := o2.(*con.Blob)

	assert.Equal(t, b.Content, "helloChanged")

	d, err = ioutil.ReadFile(helloPath)
	assert.NoError(t, err)
	assert.Equal(t, "helloChanged", string(d))

	//dummy2が存在すること
	_, ok := res.repo.i.Entries[data.EntryKey{Path: "xxx/dummy2.txt", Stage: 0}]
	assert.True(t, ok)
	stat, _ := repo.w.StatFile("xxx/dummy2.txt")
	assert.NotNil(t, stat)

	//headが変わっていること
	headObjId, err := res.repo.r.ReadHead()
	assert.NoError(t, err)
	assert.NotEqual(t, beforeResetHeadObjId, headObjId)

	//ORIG_HEADがreset前のHEADなこと
	oh, err := ioutil.ReadFile(filepath.Join(tempPath, ".git/ORIG_HEAD"))
	assert.NoError(t, err)
	str := strings.TrimSpace(string(oh))
	assert.Equal(t, beforeResetHeadObjId, str)

}
