package src

import (
	"bytes"
	"fmt"
	"io/ioutil"
	data "mygit/src/database"
	con "mygit/src/database/content"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func PrepareCherryPick(t *testing.T) string {

	// A -> B   master
	//   \
	//      C  -> D test1

	//BにDをcherryPickして、BにCの変更が含まれないかを確認
	//(hello2がtestのままで、helloがchangedでhello3があること)

	cur, err := os.Getwd()
	assert.NoError(t, err)
	tempPath, err := ioutil.TempDir(cur, "")
	assert.NoError(t, err)

	helloPath := CreateFiles(t, tempPath, "hello.txt", "test\n")
	hello2Path := CreateFiles(t, tempPath, "hello2.txt", "hello2\n")

	is := []string{tempPath}

	//A
	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit1", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartBranch(tempPath, []string{"test1"}, &BranchOption{}, &buf)
	assert.NoError(t, err)

	err = StartCheckout(tempPath, []string{"test1"}, &buf)
	assert.NoError(t, err)

	//C
	f1, err := os.Create(hello2Path)
	assert.NoError(t, err)
	f1.Write([]byte("changed2"))
	f1.Close()

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit3", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	//D
	CreateFiles(t, tempPath, "hello3.txt", "hello3\n")

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit3", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartCheckout(tempPath, []string{"master"}, &buf)
	assert.NoError(t, err)

	//B
	f2, err := os.Create(helloPath)
	assert.NoError(t, err)
	f2.Write([]byte("changed"))

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit6", &buf)
	assert.NoError(t, err)

	return tempPath
}

func TestCherryPick(t *testing.T) {
	tempPath := PrepareCherryPick(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	rev, err := ParseRev("@")
	assert.NoError(t, err)
	beforeResetHeadObjId, err := ResolveRev(rev, repo)
	assert.NoError(t, err)

	rev, err = ParseRev("test1")
	assert.NoError(t, err)
	objId, err := ResolveRev(rev, repo)
	assert.NoError(t, err)

	var buf bytes.Buffer

	err = StartCherryPick(tempPath, []string{objId}, &CherryPickOption{}, &buf)
	assert.NoError(t, err)

	newRepo := GenerateRepository(tempPath, gitPath, dbPath)

	err = newRepo.i.Load()
	assert.NoError(t, err)

	//index
	// hello3
	o := newRepo.i.Entries[data.EntryKey{Path: "hello3.txt", Stage: 0}]
	o2, err := newRepo.d.ReadObject(o.GetObjId())
	assert.NoError(t, err)
	b := o2.(*con.Blob)
	assert.Equal(t, b.Content, "hello3\n")
	// hello2
	o = newRepo.i.Entries[data.EntryKey{Path: "hello2.txt", Stage: 0}]
	o2, err = newRepo.d.ReadObject(o.GetObjId())
	assert.NoError(t, err)
	b = o2.(*con.Blob)
	assert.Equal(t, b.Content, "hello2\n")

	//workspace
	// hello3
	d, err := ioutil.ReadFile(filepath.Join(tempPath, "hello3.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "hello3\n", string(d))
	// hello2
	d, err = ioutil.ReadFile(filepath.Join(tempPath, "hello2.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "hello2\n", string(d))

	//headが変わっていること
	headObjId, err := repo.r.ReadHead()
	assert.NoError(t, err)
	assert.NotEqual(t, beforeResetHeadObjId, headObjId)

}

func PrepareCherryPickConflict(t *testing.T) string {

	// A -> B   master
	//   \
	//      C  -> D test1

	//BにDをcherryPickして、BにCの変更が含まれないかを確認(このPrepareではBのhello.txtとDのhello.txtをコンフリクトさせる)

	cur, err := os.Getwd()
	assert.NoError(t, err)
	tempPath, err := ioutil.TempDir(cur, "")
	assert.NoError(t, err)

	helloPath := CreateFiles(t, tempPath, "hello.txt", "test\n")
	hello2Path := CreateFiles(t, tempPath, "hello2.txt", "hello2\n")

	is := []string{tempPath}

	//A
	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit1", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartBranch(tempPath, []string{"test1"}, &BranchOption{}, &buf)
	assert.NoError(t, err)

	err = StartCheckout(tempPath, []string{"test1"}, &buf)
	assert.NoError(t, err)

	//C
	f1, err := os.Create(hello2Path)
	assert.NoError(t, err)
	f1.Write([]byte("changed2"))
	f1.Close()

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit3", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	//D
	CreateFiles(t, tempPath, "hello3.txt", "hello3\n")
	f3, err := os.Create(helloPath)
	assert.NoError(t, err)
	f3.Write([]byte("changedD"))

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit3", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartCheckout(tempPath, []string{"master"}, &buf)
	assert.NoError(t, err)

	//B
	f2, err := os.Create(helloPath)
	assert.NoError(t, err)
	f2.Write([]byte("changedB"))

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit6", &buf)
	assert.NoError(t, err)

	return tempPath
}

func TestCherryPickConflict(t *testing.T) {
	tempPath := PrepareCherryPickConflict(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	rev, err := ParseRev("@")
	assert.NoError(t, err)
	beforeResetHeadObjId, err := ResolveRev(rev, repo)
	assert.NoError(t, err)

	rev, err = ParseRev("test1")
	assert.NoError(t, err)
	objId, err := ResolveRev(rev, repo)
	assert.NoError(t, err)

	var buf bytes.Buffer

	err = StartCherryPick(tempPath, []string{objId}, &CherryPickOption{}, &buf)
	assert.NoError(t, err)

	newRepo := GenerateRepository(tempPath, gitPath, dbPath)

	err = newRepo.i.Load()
	assert.NoError(t, err)

	shortObjId := repo.d.ShortObjId(objId)

	expected := fmt.Sprintf(`Auto-merging hello.txt
CONFLICT (content): Merge conflict in hello.txt
error: could not apply %s... commit3
hint: 
 after resolving the conflicts, mark the corrected paths
 with 'mygit add <paths>' or 'mygit rm <paths>'
 and commit the result with 'mygit commit'
`, shortObjId)
	str := buf.String()

	//エラーメッセージが返ること
	if diff := cmp.Diff(expected, str); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}

	//headが変わっていないこと
	headObjId, err := repo.r.ReadHead()
	assert.NoError(t, err)
	assert.Equal(t, beforeResetHeadObjId, headObjId)

	//.git/CHERRY_PICK_HEADがあること
	stat, _ := repo.w.StatFile(".git/CHERRY_PICK_HEAD")
	assert.NotNil(t, stat)

}

func PrepareAfterCherryPickConflict(t *testing.T) string {

	// A -> B   master
	//   \
	//      C  -> D test1

	//BにDをcherryPickして、BにCの変更が含まれないかを確認(このPrepareではBのhello.txtとDのhello.txtをコンフリクトさせる)

	cur, err := os.Getwd()
	assert.NoError(t, err)
	tempPath, err := ioutil.TempDir(cur, "")
	assert.NoError(t, err)

	helloPath := CreateFiles(t, tempPath, "hello.txt", "test\n")
	hello2Path := CreateFiles(t, tempPath, "hello2.txt", "hello2\n")

	is := []string{tempPath}

	//A
	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit1", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartBranch(tempPath, []string{"test1"}, &BranchOption{}, &buf)
	assert.NoError(t, err)

	err = StartCheckout(tempPath, []string{"test1"}, &buf)
	assert.NoError(t, err)

	//C
	f1, err := os.Create(hello2Path)
	assert.NoError(t, err)
	f1.Write([]byte("changed2"))
	f1.Close()

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit3", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	//D
	CreateFiles(t, tempPath, "hello3.txt", "hello3\n")
	f3, err := os.Create(helloPath)
	assert.NoError(t, err)
	f3.Write([]byte("changedD"))

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit3", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartCheckout(tempPath, []string{"master"}, &buf)
	assert.NoError(t, err)

	//B
	f2, err := os.Create(helloPath)
	assert.NoError(t, err)
	f2.Write([]byte("changedB"))

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit6", &buf)
	assert.NoError(t, err)

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	rev, err := ParseRev("test1")
	assert.NoError(t, err)
	objId, err := ResolveRev(rev, repo)
	assert.NoError(t, err)

	err = StartCherryPick(tempPath, []string{objId}, &CherryPickOption{}, &buf)
	assert.NoError(t, err)

	return tempPath
}

func TestCherryPickConflictMessageInProgress(t *testing.T) {
	tempPath := PrepareAfterCherryPickConflict(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	rev, err := ParseRev("test1")
	assert.NoError(t, err)
	objId, err := ResolveRev(rev, repo)
	assert.NoError(t, err)

	var buf bytes.Buffer

	err = StartCherryPick(tempPath, []string{objId}, &CherryPickOption{}, &buf)
	assert.NoError(t, err)

	newRepo := GenerateRepository(tempPath, gitPath, dbPath)

	err = newRepo.i.Load()
	assert.NoError(t, err)

	shortObjId := repo.d.ShortObjId(objId)

	expected := fmt.Sprintf(`error: could not apply %s... commit3
hint: 
 after resolving the conflicts, mark the corrected paths
 with 'mygit add <paths>' or 'mygit rm <paths>'
 and commit the result with 'mygit commit'
`, shortObjId)
	str := buf.String()

	//エラーメッセージが返ること
	if diff := cmp.Diff(expected, str); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}

}

func TestCherryPickContinueWithoutAddingIndex(t *testing.T) {
	tempPath := PrepareAfterCherryPickConflict(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	rev, err := ParseRev("test1")
	assert.NoError(t, err)
	objId, err := ResolveRev(rev, repo)
	assert.NoError(t, err)

	var buf bytes.Buffer

	err = StartCherryPick(tempPath, []string{objId}, &CherryPickOption{hasContinue: true}, &buf)
	assert.NoError(t, err)

	newRepo := GenerateRepository(tempPath, gitPath, dbPath)

	err = newRepo.i.Load()
	assert.NoError(t, err)

	expected := `error: Commiting is not possible because you have unmerged files
hint: Fix them up in the work tree, and then use 'mygit add/rm <file>'
hint: as appropriate to mark resolution and make a commit.
fatal: Exiting because of an unresolved conflict.
`
	str := buf.String()

	//エラーメッセージが返ること
	if diff := cmp.Diff(expected, str); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}

}
