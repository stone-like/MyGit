package src

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func PrepareRevert(t *testing.T) string {

	// A -> B -> C -> D -> E -> F -> G master

	cur, err := os.Getwd()
	assert.NoError(t, err)
	tempPath, err := ioutil.TempDir(cur, "")
	assert.NoError(t, err)

	aPath := CreateFiles(t, tempPath, "a.txt", "1\n")

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

	//B
	f1, err := os.Create(aPath)
	assert.NoError(t, err)
	f1.Write([]byte("2\n"))
	f1.Close()

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit2", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	//C
	f2, err := os.Create(aPath)
	assert.NoError(t, err)
	f2.Write([]byte("3\n"))
	f2.Close()

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit3", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	//D
	bPath := CreateFiles(t, tempPath, "b.txt", "4\n")

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit4", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	//E
	f3, err := os.Create(aPath)
	assert.NoError(t, err)
	f3.Write([]byte("5\n"))
	f3.Close()

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit5", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	//F
	f4, err := os.Create(bPath)
	assert.NoError(t, err)
	f4.Write([]byte("6\n"))
	f4.Close()

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit6", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	//G
	f5, err := os.Create(bPath)
	assert.NoError(t, err)
	f5.Write([]byte("7\n"))
	f5.Close()

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit7", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	return tempPath
}

func TestRevert(t *testing.T) {
	tempPath := PrepareRevert(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	// gitPath := filepath.Join(tempPath, ".git")
	// dbPath := filepath.Join(gitPath, "objects")
	// repo := GenerateRepository(tempPath, gitPath, dbPath)

	var buf bytes.Buffer
	err := StartRevert(tempPath, []string{"@^^"}, &SequenceOption{}, &buf)
	assert.NoError(t, err)

	//Eのa.txt -> 5が打ち消されていることを確認
	//b.txt = 7,a.txt = 3

	content, err := ioutil.ReadFile(filepath.Join(tempPath, "a.txt"))
	assert.NoError(t, err)
	expected := "3\n"
	if diff := cmp.Diff(expected, string(content)); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}

	content, err = ioutil.ReadFile(filepath.Join(tempPath, "b.txt"))
	assert.NoError(t, err)
	expected = "7\n"
	if diff := cmp.Diff(expected, string(content)); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}

}

func TestRevertConflict(t *testing.T) {
	tempPath := PrepareRevert(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	rev, err := ParseRev("@^")
	assert.NoError(t, err)
	fObjId, err := ResolveRev(rev, repo)
	assert.NoError(t, err)
	shortFObjId := repo.d.ShortObjId(fObjId)

	rev, err = ParseRev("@^^")
	assert.NoError(t, err)
	eObjId, err := ResolveRev(rev, repo)
	assert.NoError(t, err)
	shortEObjId := repo.d.ShortObjId(eObjId)

	var buf bytes.Buffer
	err = StartRevert(tempPath, []string{"@^", "@^^"}, &SequenceOption{}, &buf)
	assert.NoError(t, err)

	//F- > G
	//  b.txt -> 7
	//F -> E
	// b.txt  -> 4
	//でmod/modコンフリクト

	str := buf.String()
	expected := fmt.Sprintf("Auto-merging b.txt\nCONFLICT (content): Merge conflict in b.txt\nerror: could not apply parent of %s... commit6\nhint: \n after resolving the conflicts, mark the corrected paths\n with 'mygit add <paths>' or 'mygit rm <paths>'\n and commit the result with 'mygit commit'\n", shortFObjId)
	if diff := cmp.Diff(expected, str); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}

	//todoに未revertのF,Eがあることを確認
	seqPath := filepath.Join(repo.r.Path, "sequencer")
	todoPath := filepath.Join(seqPath, "todo")
	// abortPath := filepath.Join(seqPath, "abort-safety")
	// headPath := filepath.Join(seqPath, "head")

	con, err := ioutil.ReadFile(todoPath)
	assert.NoError(t, err)

	str2 := string(con)
	expected2 := fmt.Sprintf("revert %s commit6\nrevert %s commit5\n", shortFObjId, shortEObjId)
	if diff := cmp.Diff(expected2, str2); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}
}

//連続revertとcontinueだけtestする
func PrepareRevertConflict(t *testing.T) string {

	// A -> B -> C -> D -> E -> F -> G master

	cur, err := os.Getwd()
	assert.NoError(t, err)
	tempPath, err := ioutil.TempDir(cur, "")
	assert.NoError(t, err)

	aPath := CreateFiles(t, tempPath, "a.txt", "1\n")

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

	//B
	f1, err := os.Create(aPath)
	assert.NoError(t, err)
	f1.Write([]byte("2\n"))
	f1.Close()

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit2", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	//C
	f2, err := os.Create(aPath)
	assert.NoError(t, err)
	f2.Write([]byte("3\n"))
	f2.Close()

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit3", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	//D
	bPath := CreateFiles(t, tempPath, "b.txt", "4\n")

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit4", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	//E
	f3, err := os.Create(aPath)
	assert.NoError(t, err)
	f3.Write([]byte("5\n"))
	f3.Close()

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit5", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	//F
	f4, err := os.Create(bPath)
	assert.NoError(t, err)
	f4.Write([]byte("6\n"))
	f4.Close()

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit6", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	//G
	f5, err := os.Create(bPath)
	assert.NoError(t, err)
	f5.Write([]byte("7\n"))
	f5.Close()

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit7", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartRevert(tempPath, []string{"@^", "@^^"}, &SequenceOption{}, &buf)
	assert.NoError(t, err)

	return tempPath
}

func TestRevertContinue(t *testing.T) {
	tempPath := PrepareRevertConflict(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	//コンフリクトを解消
	f, err := os.Create(filepath.Join(tempPath, "b.txt"))
	assert.NoError(t, err)
	f.Write([]byte("4\n"))
	f.Close()

	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)

	// rev, err := ParseRev("@^")
	// assert.NoError(t, err)
	// fObjId, err := ResolveRev(rev, repo)
	// assert.NoError(t, err)
	// shortFObjId := repo.d.ShortObjId(fObjId)

	var buf bytes.Buffer
	err = StartRevert(tempPath, []string{"@^"}, &SequenceOption{hasContinue: true}, &buf)
	assert.NoError(t, err)

	//sequncerファイルが消えていること
	seqPath := filepath.Join(repo.r.Path, "sequencer")
	todoPath := filepath.Join(seqPath, "todo")
	abortPath := filepath.Join(seqPath, "abort-safety")
	headPath := filepath.Join(seqPath, "head")

	stat, _ := os.Stat(todoPath)
	assert.Nil(t, stat)
	stat, _ = os.Stat(abortPath)
	assert.Nil(t, stat)
	stat, _ = os.Stat(headPath)
	assert.Nil(t, stat)

	//F,Eと二つrevert済みなこと
	//Fのrevertはconflictしたが、上でコンフリクトをb.txt = 4として解消,Eをrevrtすることでa.txtが5 -> 3になっていてほしい
	con1, err := ioutil.ReadFile(filepath.Join(tempPath, "a.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "3\n", string(con1))
	con2, err := ioutil.ReadFile(filepath.Join(tempPath, "b.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "4\n", string(con2))

}
